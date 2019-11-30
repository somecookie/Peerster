package blockchain

//TxPublish represents a transaction to be published. When a new file is
//indexed (and potentially shared) the others peers need to be notified
//with a corresponding transaction.
//Name         string
//Size         int64 is the size in bytes
//MetafileHash []byte
type TxPublish struct {
	Name         string
	Size         int64
	MetafileHash []byte
}

type BlockPublish struct{
	PrevHash [32]byte
	Transaction TxPublish
}