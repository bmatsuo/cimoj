// Code generated by "stringer -type=ItemType"; DO NOT EDIT

package main

import "fmt"

const _ItemType_name = "ItemMoneyXXSItemMoneyXSItemMoneySMItemMoneyMDItemMoneyLGItemMoneyXLItemMoneyXXLItemPoisonItemRowClearItemPushUpItemBulletItemScrambleItemRecolor"

var _ItemType_index = [...]uint8{0, 12, 23, 34, 45, 56, 67, 79, 89, 101, 111, 121, 133, 144}

func (i ItemType) String() string {
	if i >= ItemType(len(_ItemType_index)-1) {
		return fmt.Sprintf("ItemType(%d)", i)
	}
	return _ItemType_name[_ItemType_index[i]:_ItemType_index[i+1]]
}
