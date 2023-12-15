import asyncio
import json
import typing
import uuid
from asyncio import Task, CancelledError
from dataclasses import dataclass, asdict

import aiohttp
from aiohttp.web_ws import WebSocketResponse

from app.base.accessor import BaseAccessor
from app.base.utils import do_by_timeout_wrapper

if typing.TYPE_CHECKING:
    from app.base.application import Request


@dataclass
class Event:
    kind: int

    def __str__(self):
        return f'Event<{self.kind}>'


class WSContext:
    def __init__(
            self,
            accessor: 'WSAccessor',
            request: 'Request',
            close_callback: typing.Callable[[str], typing.Awaitable] | None = None,
    ):
        self._accessor = accessor
        self._request = request
        self.connection_id: typing.Optional[str] = None
        self._close_callback = close_callback

    async def __aenter__(self) -> str:
        self.connection_id = await self._accessor.open(self._request, close_callback=self._close_callback)
        return self.connection_id

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self._accessor.close(self.connection_id)


@dataclass
class Connection:
    session: WebSocketResponse
    ip_addr: str
    timeout_task: Task
    close_callback: typing.Callable[[str], typing.Awaitable] | None


class WSAccessor(BaseAccessor):
    class Meta:
        name = 'ws_accessor'

    CONNECTION_TIMEOUT_SECONDS = 60

    def _init_(self) -> None:
        self._connections: dict[str, Connection] = {}

    async def open(
            self,
            request: 'Request',
            close_callback: typing.Callable[[str], typing.Awaitable[typing.Any]] | None = None,
    ) -> str:
        ws_response = WebSocketResponse(
            receive_timeout=self.CONNECTION_TIMEOUT_SECONDS,
            autoping=False
        )
        await ws_response.prepare(request)
        connection_id = str(uuid.uuid4())

        self.logger.info(f'Handling new connection with {connection_id=}')

        self._connections[connection_id] = Connection(
            session=ws_response,
            timeout_task=self._create_timeout_task(connection_id),
            close_callback=close_callback,
            ip_addr=request.remote,
        )
        return connection_id

    def _create_timeout_task(self, connection_id: str) -> Task:
        def log_timeout(result: Task):
            try:
                exc = result.exception()
            except CancelledError:
                return

            if exc:
                self.logger.error('Can not close connection by timeout', exc_info=result.exception())
            else:
                self.logger.info(f'Connection with {connection_id=} was closed by inactivity')

        task = asyncio.create_task(
            do_by_timeout_wrapper(
                self.close,
                self.CONNECTION_TIMEOUT_SECONDS,
                args=[connection_id],
            )
        )
        task.add_done_callback(log_timeout)
        return task

    async def close(self, connection_id: str):
        connection = self._connections.pop(connection_id, None)
        if not connection:
            return

        self.logger.info(f'Closing {connection_id=}')

        if connection.close_callback:
            await connection.close_callback(connection_id)

        if not connection.session.closed:
            await connection.session.close()

    async def stream(self, connection_id: str) -> typing.AsyncIterable[Event]:
        async for message in self._connections[connection_id].session:
            if message.type == aiohttp.WSMsgType.PING:
                await self._connections[connection_id].session.pong()
                await self.refresh_connection(connection_id)
            yield Event(kind=message.type)

    async def initial_info(self, connection_id: str) -> typing.Tuple[str, str]:
        return (self._connections[connection_id].ip_addr,
                (await self._connections[connection_id].session.receive_bytes(timeout=10)).hex())

    async def refresh_connection(self, connection_id: str):
        self._connections[connection_id].timeout_task.cancel()
        self._connections[connection_id].timeout_task = self._create_timeout_task(connection_id)
