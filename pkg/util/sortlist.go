package util

import (
	"container/list"
)

type SortedLinkedList struct {
	*list.List
	compareFunc func(old, new interface{}) bool
}

func NewSortedList(compare func(old, new interface{}) bool) *SortedLinkedList {
	return &SortedLinkedList{list.New(), compare}
}

func (this *SortedLinkedList) findInsertPlaceElement(value interface{}) *list.Element {
	for element := this.Front(); element != nil; element = element.Next() {
		tempValue := element.Value
		if this.compareFunc(tempValue, value) {
			return element
		}
	}
	return nil
}

func (this *SortedLinkedList) PutOnTop(value interface{}) {
	if this.List.Len() == 0 {
		this.PushFront(value)
		return
	}
	// 在最后
	if this.compareFunc(value, this.Back().Value) {
		this.PushBack(value)
		return
	}
	// 在最前
	if this.compareFunc(this.List.Front().Value, value) {
		this.PushFront(value)
		return
	}
	// 在中间
	if this.compareFunc(this.List.Back().Value, value) && this.compareFunc(value, this.Front().Value) {
		element := this.findInsertPlaceElement(value)
		if element != nil {
			this.InsertBefore(value, element)
		}
		return
	}
	// 以上规则都不存在
	this.PushBack(value)
}
