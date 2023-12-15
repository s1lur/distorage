import logging

import aiohttp
from app.base.accessor import BaseManager
from app.store.ws.ws_accessor import Event


class NodeManager(BaseManager):
    class Meta:
        name = 'node_manager'

    async def handle(self, connection_id: str):
        logging.info(f'accepting new connection: {connection_id}')
        ip_addr, pub_addr = await self.store.ws_accessor.initial_info(connection_id)
        await self.store.nodes_accessor.add(
            _id=connection_id,
            pub_addr=pub_addr,
            ip_addr=ip_addr,
        )
        async for _ in self.store.ws_accessor.stream(connection_id):
            continue

    async def on_user_disconnect(self, connection_id: str) -> None:
        await self.store.nodes_accessor.remove(connection_id)
