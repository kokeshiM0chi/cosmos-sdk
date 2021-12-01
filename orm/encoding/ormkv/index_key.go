package ormkv

import (
	"bytes"
	"io"

	"github.com/cosmos/cosmos-sdk/orm/types/ormerrors"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// IndexKeyCodec is the codec for (non-unique) index keys.
type IndexKeyCodec struct {
	*KeyCodec
	tableName    protoreflect.FullName
	pkFieldOrder []int
}

var _ IndexCodec = &IndexKeyCodec{}

// NewIndexKeyCodec creates a new IndexKeyCodec.
func NewIndexKeyCodec(prefix []byte, messageDescriptor protoreflect.MessageDescriptor, indexFields, primaryKeyFields []protoreflect.Name) (*IndexKeyCodec, error) {
	indexFieldMap := map[protoreflect.Name]int{}

	keyFields := make([]protoreflect.Name, 0, len(indexFields)+len(primaryKeyFields))
	for i, f := range indexFields {
		indexFieldMap[f] = i
		keyFields = append(keyFields, f)
	}

	numIndexFields := len(indexFields)
	numPrimaryKeyFields := len(primaryKeyFields)
	pkFieldOrder := make([]int, numPrimaryKeyFields)
	k := 0
	for j, f := range primaryKeyFields {
		if i, ok := indexFieldMap[f]; ok {
			pkFieldOrder[j] = i
			continue
		}
		keyFields = append(keyFields, f)
		pkFieldOrder[j] = numIndexFields + k
		k++
	}

	cdc, err := NewKeyCodec(prefix, messageDescriptor, keyFields)
	if err != nil {
		return nil, err
	}

	return &IndexKeyCodec{
		KeyCodec:     cdc,
		pkFieldOrder: pkFieldOrder,
		tableName:    messageDescriptor.FullName(),
	}, nil
}

func (cdc IndexKeyCodec) DecodeIndexKey(k, _ []byte) (indexFields, primaryKey []protoreflect.Value, err error) {

	values, err := cdc.Decode(bytes.NewReader(k))
	// got prefix key
	if err == io.EOF {
		return values, nil, nil
	} else if err != nil {
		return nil, nil, err
	}

	// got prefix key
	if len(values) < len(cdc.fieldCodecs) {
		return values, nil, nil
	}

	numPkFields := len(cdc.pkFieldOrder)
	pkValues := make([]protoreflect.Value, numPkFields)

	for i := 0; i < numPkFields; i++ {
		pkValues[i] = values[cdc.pkFieldOrder[i]]
	}

	return values, pkValues, nil
}


func (cdc IndexKeyCodec) DecodeEntry(k, v []byte) (Entry, error) {
	idxValues, pk, err := cdc.DecodeIndexKey(k, v)
	if err != nil {
		return nil, err
	}

	return &IndexKeyEntry{
		TableName:   cdc.tableName,
		Fields:      cdc.fieldNames,
		IndexValues: idxValues,
		PrimaryKey:  pk,
	}, nil
}

func (i IndexKeyCodec) EncodeEntry(entry Entry) (k, v []byte, err error) {
	indexEntry, ok := entry.(*IndexKeyEntry)
	if !ok {
		return nil, nil, ormerrors.BadDecodeEntry
	}

	if indexEntry.TableName != i.tableName {
		return nil, nil, ormerrors.BadDecodeEntry
	}

	bz, err := i.KeyCodec.Encode(indexEntry.IndexValues)
	if err != nil {
		return nil, nil, err
	}

	return bz, sentinel, nil
}

var sentinel = []byte{0}

func (cdc IndexKeyCodec) EncodeKVFromMessage(message protoreflect.Message) (k, v []byte, err error) {
	_, k, err = cdc.EncodeFromMessage(message)
	return k, sentinel, err
}
