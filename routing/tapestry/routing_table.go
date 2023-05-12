/*
 *  Brown University, CS138, Spring 2023
 *
 *  Purpose: Defines the RoutingTable type and provides methods for interacting
 *  with it.
 */

package tapestry

import (
	"sync"
)

// RoutingTable has a number of levels equal to the number of digits in an ID
// (default 40). Each level has a number of slots equal to the digit base
// (default 16). A node that exists on level n thereby shares a prefix of length
// n with the local node. Access to the routing table is protected by a mutex.
type RoutingTable struct {
	localId ID                 // The ID of the local tapestry node
	Rows    [DIGITS][BASE][]ID // The rows of the routing table (stores IDs of remote tapestry nodes)
	mutex   sync.Mutex         // To manage concurrent access to the routing table (could also have a per-level mutex)
}

// NewRoutingTable creates and returns a new routing table, placing the local node at the
// appropriate slot in each level of the table.
func NewRoutingTable(me ID) *RoutingTable {
	t := new(RoutingTable)
	t.localId = me

	// Create the node lists with capacity of SLOTSIZE
	for i := 0; i < DIGITS; i++ {
		for j := 0; j < BASE; j++ {
			t.Rows[i][j] = make([]ID, 0, SLOTSIZE)
		}
	}

	// Make sure each row has at least our node in it
	for i := 0; i < DIGITS; i++ {
		slot := t.Rows[i][t.localId[i]]
		t.Rows[i][t.localId[i]] = append(slot, t.localId)
	}

	return t
}

// Add adds the given node to the routing table.
//
// Note you should not add the node to preceding levels. You need to add the node
// to one specific slot in the routing table (or replace an element if the slot is full
// at SLOTSIZE).
//
// Returns true if the node did not previously exist in the table and was subsequently added.
// Returns the previous node in the table, if one was overwritten.
func (t *RoutingTable) Add(remoteNodeId ID) (added bool, previous *ID) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	// if its trying to add itself
	if t.localId.String() == remoteNodeId.String() {
		return false, nil
	}

	rowIdx := SharedPrefixLength(t.localId, remoteNodeId)
	// can be commented out
	if rowIdx == DIGITS {
		return false, nil
	}
	slot := t.Rows[rowIdx][remoteNodeId[rowIdx]]

	// check if the node already exists in the table
	for i := 0; i < len(slot); i++ {
		if slot[i].String() == remoteNodeId.String() {
			return false, nil
		}
	}
	// if not exist, add it
	// we update the slot and find out who is the further, the lastPlace(the furthest)
	// may be removed or appended at the end of of the slot
	lastPlace := remoteNodeId
	for i := 0; i < len(slot); i++ {
		if t.localId.Closer(lastPlace, slot[i]) {
			temp := slot[i]
			slot[i] = lastPlace
			lastPlace = temp
		}
	}

	// if the slot not full yet
	if len(slot) < SLOTSIZE {
		t.Rows[rowIdx][remoteNodeId[rowIdx]] = append(t.Rows[rowIdx][remoteNodeId[rowIdx]], lastPlace)
		return true, nil
	} else {
		// slot is already full, we need to drop the lastPlace
		t.Rows[rowIdx][remoteNodeId[rowIdx]] = slot
		if lastPlace.String() == remoteNodeId.String() {
			return false, nil
		} else {
			return true, &lastPlace
		}
	}

	// TODO(students): [Tapestry] Implement me!

}

// Remove removes the specified node from the routing table, if it exists.
// Returns true if the node was in the table and was successfully removed.
// Return false if a node tries to remove itself from the table.
func (t *RoutingTable) Remove(remoteNodeId ID) (wasRemoved bool) {

	t.mutex.Lock()
	defer t.mutex.Unlock()
	// how dare you try removing yourself
	if t.localId.String() == remoteNodeId.String() {
		return false
	}
	rowIdx := SharedPrefixLength(t.localId, remoteNodeId)
	// can be commented out
	if rowIdx == DIGITS {
		return false
	}
	//  locate the right slot
	slot := t.Rows[rowIdx][remoteNodeId[rowIdx]]
	// newslot := make([]ID, 0, SLOTSIZE)
	for i := 0; i < len(slot); i++ {
		if slot[i] == remoteNodeId {
			slot = append(slot[:i], slot[i+1:]...)
			// update the slot
			t.Rows[rowIdx][remoteNodeId[rowIdx]] = slot
			return true
		}
	}

	// TODO(students): [Tapestry] Implement me!
	return false
}

// GetLevel gets ALL nodes on the specified level of the routing table, EXCLUDING the local node.
func (t *RoutingTable) GetLevel(level int) (nodeIds []ID) {

	t.mutex.Lock()
	defer t.mutex.Unlock()
	row := t.Rows[level]
	for i := 0; i < BASE; i++ {
		slot := row[i]
		for j := 0; j < len(slot); j++ {
			nodeId := slot[j]
			if nodeId.String() != t.localId.String() {
				nodeIds = append(nodeIds, nodeId)
			}
		}
	}

	// TODO(students): [Tapestry] Implement me!
	return
}

// FindNextHop searches the table for the closest next-hop node for the provided ID starting at the given level.
func (t *RoutingTable) FindNextHop(id ID, level int32) ID {

	// // exceed level (base case): then we are the root and return local ID
	// if level >= DIGITS {
	// 	return t.localId
	// }

	// // look right (and wrap around) until we find the first non-empty slot
	// slotIdx := id[level]
	// slot := t.Rows[level][slotIdx]
	// for {
	// 	if len(slot) > 0 {
	// 		break
	// 	}
	// 	slotIdx = (slotIdx + 1) % BASE
	// 	slot = t.Rows[level][slotIdx]
	// }

	// // if first node in slot is local
	// // go to next level and start at slot id[level+1]
	// if slot[0].String() == t.localId.String() {
	// 	return t.FindNextHop(id, level+1)
	// }

	// // if first node in slot is non-local, return it
	// return slot[0]
	//new try
	for curLevel := level; curLevel < DIGITS; curLevel++ {
		curDigit := id[curLevel]
		cont := true
		// at this level, look at the right [curslot, Base]
		for j := curDigit; j < BASE; j++ {
			curSlot := t.Rows[curLevel][j]
			if len(curSlot) > 0 {
				if curSlot[0].String() != t.localId.String() {
					return curSlot[0]
				} else {
					// if localID = curslot[0], we go to next level and cmp the next digit, and we dont need to look at the left
					cont = false
					break
				}
			}
		}
		// now we look at the same level but the left part and see if there s a nonempty slot
		if cont {
			for j := 0; j < int(curDigit); j++ {
				curSlot := t.Rows[curLevel][j]
				if len(curSlot) > 0 {
					if curSlot[0].String() != t.localId.String() {
						return curSlot[0]
					} else {
						// if localID = curslot[0], we go to next level and cmp the next digit, and we dont need to look at the left
						break
					}

				}
			}

		}

	}
	return t.localId
}