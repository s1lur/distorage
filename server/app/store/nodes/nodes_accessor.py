from dataclasses import dataclass

from app.base.accessor import BaseAccessor


@dataclass
class Node:
    id: str
    pub_addr: str
    ip_addr: str

    def __str__(self):
        return f'Node {self.pubAddr} ({self.ip_addr})'


class NodesAccessor(BaseAccessor):
    def _init_(self) -> None:
        self._nodes: dict[str, Node] = {}

    async def list_nodes(self) -> list[Node]:
        return list(self._nodes.values())

    async def node_dict(self) -> dict[str, str]:
        return {node.pub_addr: node.ip_addr for node in self._nodes.values()}

    async def add(
            self,
            _id: str,
            pub_addr: str,
            ip_addr: str,
    ) -> Node:
        node = Node(
            id=_id,
            pub_addr=pub_addr,
            ip_addr=ip_addr,
        )
        self._nodes[_id] = node
        return node

    async def remove(self, _id: str) -> None:
        self._nodes.pop(_id)

    async def get(self, _id: str) -> Node:
        return self._nodes[_id]
