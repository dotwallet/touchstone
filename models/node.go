package models

const (
	TABLE_NODE = "node"
)

type Node struct {
	Host   string `bson:"host"`
	Pubkey string `bson:"pubkey"`
}

type NodeRepository struct {
	db *MongoDb
}

func (this *NodeRepository) GetNodes(offset int, limit int) ([]*Node, error) {
	nodes := make([]*Node, 0, 8)
	err := this.db.GetMany(TABLE_NODE, nil, nil, MONGO_ID, offset, limit, nodes)
	return nodes, err
}

func (this *NodeRepository) AddNode(Host string) error {
	node := &Node{
		Host: Host,
	}
	return this.db.Insert(TABLE_NODE, node)
}
