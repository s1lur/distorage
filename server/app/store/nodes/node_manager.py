import aiohttp
from app.base.accessor import BaseManager
from app.store.ws.ws_accessor import Event


class NodeManager(BaseManager):
    class Meta:
        name = 'geo_manager'

    MAX_ERROR = 0.05

    async def handle(self, connection_id: str):
        ip_addr, pub_addr = await self.store.ws_accessor.initial_info(connection_id)
        await self.store.nodes_accessor.add(
            _id=connection_id,
            pub_addr=pub_addr,
            ip_addr=ip_addr,
        )
        async for event in self.store.ws_accessor.stream(connection_id):
            should_continue = await self._handle_event(event, connection_id)
            if not should_continue:
                break

    async def _handle_event(self, event: Event, connection_id: str) -> bool:
        if event.kind == aiohttp.WSMsgType.PING:
            return True
        elif event.kind == aiohttp.WSMsgType.CLOSE:
            user = await self.store.nodes_accessor.get(_id=connection_id)
            self.logger.info(f'User {user} disconnected')
            await self.on_user_disconnect(connection_id)
            return False
        else:
            return False

    async def on_user_disconnect(self, connection_id: str) -> None:
        await self.store.nodes_accessor.remove(connection_id)
