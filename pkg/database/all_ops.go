/*
Copyright 2019-2020 vChain, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package database

import (
	"crypto/sha256"

	"github.com/codenotary/immudb/embedded/store"
	"github.com/codenotary/immudb/pkg/api/schema"
)

// ExecAll like Set it permits many insertions at once.
// The difference is that is possible to to specify a list of a mix of key value set and zAdd insertions.
// If zAdd reference is not yet present on disk it's possible to add it as a regular key value and the reference is done onFly
func (d *db) ExecAll(req *schema.ExecAllRequest) (*schema.TxMetadata, error) {
	if req == nil {
		return nil, store.ErrIllegalArguments
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	lastTxID, _ := d.st.Alh()
	d.WaitForIndexingUpto(lastTxID)

	snap, err := d.st.SnapshotSince(lastTxID)
	if err != nil {
		return nil, err
	}
	defer snap.Close()

	entries := make([]*store.KV, len(req.Operations))

	// In order to:
	// * make a memory efficient check system for keys that need to be referenced
	// * store the index of the future persisted zAdd referenced entries
	// we build a map in which we store sha256 sum as key and the index as value
	kmap := make(map[[sha256.Size]byte]bool)

	for i, op := range req.Operations {
		if op == nil {
			return nil, store.ErrIllegalArguments
		}

		kv := &store.KV{}

		switch x := op.Operation.(type) {

		case *schema.Op_Kv:
			kmap[sha256.Sum256(x.Kv.Key)] = true

			kv = EncodeKV(x.Kv.Key, x.Kv.Value)

		case *schema.Op_Ref:
			// check key does not exists or it's already a reference
			entry, err := d.getAt(EncodeKey(x.Ref.Key), x.Ref.AtTx, 0, snap, d.tx1)
			if err != nil && err != store.ErrKeyNotFound {
				return nil, err
			}
			if entry != nil && entry.ReferencedBy == nil {
				return nil, ErrFinalKeyCannotBeConvertedIntoReference
			}

			// reference arguments are converted in regular key value items and then atomically inserted
			_, exists := kmap[sha256.Sum256(x.Ref.ReferencedKey)]

			if !exists || x.Ref.AtTx > 0 {
				// check referenced key exists and it's not a reference
				refEntry, err := d.getAt(EncodeKey(x.Ref.ReferencedKey), x.Ref.AtTx, 0, snap, d.tx1)
				if err != nil {
					return nil, err
				}
				if refEntry.ReferencedBy != nil {
					return nil, ErrReferencedKeyCannotBeAReference
				}
			}

			kv = EncodeReference(x.Ref.Key, x.Ref.ReferencedKey, x.Ref.AtTx)

		case *schema.Op_ZAdd:
			// zAdd arguments are converted in regular key value items and then atomically inserted
			_, exists := kmap[sha256.Sum256(x.ZAdd.Key)]

			if !exists || x.ZAdd.AtTx > 0 {
				// check referenced key exists and it's not a reference
				refEntry, err := d.getAt(EncodeKey(x.ZAdd.Key), x.ZAdd.AtTx, 0, snap, d.tx1)
				if err != nil {
					return nil, err
				}
				if refEntry.ReferencedBy != nil {
					return nil, ErrReferencedKeyCannotBeAReference
				}
			}

			kv = EncodeZAdd(x.ZAdd.Set, x.ZAdd.Score, x.ZAdd.Key, x.ZAdd.AtTx)

		default:
			return nil, store.ErrIllegalArguments
		}

		entries[i] = kv
	}

	txMetatadata, err := d.st.Commit(entries)
	if err != nil {
		return nil, err
	}

	return schema.TxMetatadaTo(txMetatadata), nil
}
