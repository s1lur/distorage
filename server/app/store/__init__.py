import typing

from app.store.nodes import NodeManager, NodesAccessor
from app.store.ws import WSAccessor

if typing.TYPE_CHECKING:
    from app.base.application import Application


class Store:
    def __init__(self, app: "Application"):
        self.app = app
        self.ws_accessor = WSAccessor(self)
        self.node_manager = NodeManager(self)
        self.nodes_accessor = NodesAccessor(self)
