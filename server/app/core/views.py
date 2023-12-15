import os

from aiohttp import web

from app import BASE_DIR
from app.base.application import View
from app.store.ws.ws_accessor import WSContext


class IndexView(View):
    async def get(self):
        return web.json_response(await self.store.nodes_accessor.node_dict())


class WSConnectView(View):
    async def get(self):
        async with WSContext(
                accessor=self.store.ws_accessor,
                request=self.request,
                close_callback=self.store.node_manager.on_user_disconnect,
        ) as connection_id:
            await self.store.node_manager.handle(connection_id)
        return
