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
	"errors"
	"fmt"

	"github.com/codenotary/immudb/embedded/store"
	"github.com/codenotary/immudb/pkg/api/schema"
)

var ErrReferencedKeyCannotBeAReference = errors.New("referenced key cannot be a reference")
var ErrFinalKeyCannotBeConvertedIntoReference = errors.New("final key cannot be converted into a reference")

//Reference ...
func (d *db) SetReference(req *schema.ReferenceRequest) (*schema.TxMetadata, error) {
	if req == nil || len(req.Key) == 0 || len(req.ReferencedKey) == 0 {
		return nil, store.ErrIllegalArguments
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	lastTxID, _ := d.st.Alh()
	d.WaitForIndexingUpto(lastTxID)

	// check key does not exists or it's already a reference
	entry, err := d.getAt(EncodeKey(req.Key), req.AtTx, 0, d.st, d.tx1)
	if err != nil && err != store.ErrKeyNotFound {
		return nil, err
	}
	if entry != nil && entry.ReferencedBy == nil {
		return nil, ErrFinalKeyCannotBeConvertedIntoReference
	}

	// check referenced key exists and it's not a reference
	refEntry, err := d.getAt(EncodeKey(req.ReferencedKey), req.AtTx, 0, d.st, d.tx1)
	if err != nil {
		return nil, err
	}
	if refEntry.ReferencedBy != nil {
		return nil, ErrReferencedKeyCannotBeAReference
	}

	meta, err := d.st.Commit([]*store.KV{EncodeReference(req.Key, req.ReferencedKey, req.AtTx)})
	if err != nil {
		return nil, err
	}

	return schema.TxMetatadaTo(meta), err
}

//SafeReference ...
func (d *db) VerifiableSetReference(req *schema.VerifiableReferenceRequest) (*schema.VerifiableTx, error) {
	return nil, fmt.Errorf("Functionality not yet supported: %s", "VerifiableSetReference")
}
