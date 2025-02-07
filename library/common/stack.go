package common

import "container/list"

// Stack is a data structure that stores elements in a Last-In-First-Out
type Stack struct {
	list *list.List
}

// NewStack creates a new empty stack.
//
// The returned stack is empty and has a length of 0.
//
// The stack is implemented using the container/list package, which provides
// a doubly-linked list. The list is used as a stack, with the Push method
// adding elements to the front of the list and the Pop method removing
// elements from the front of the list.
//
// The stack is thread-safe, as the container/list package is thread-safe.
func NewStack() *Stack {
	return &Stack{
		list: list.New(),
	}
}

// Push adds an element to the stack and places it at the top of the stack.
//
// The element can be of any type. The element is added to the front of the
// list, so it will be the first element in the list and the top element of
// the stack when retrieved.
//
// The element is not copied, so the element added to the stack is the same
// as the one passed to the Push method.
func (s *Stack) Push(v any) {
	s.list.PushFront(v)
}

// PopRead returns the element at the top of the stack without removing it.
// If the stack is empty, it returns nil.
//
// PopRead is useful for inspecting the top element of the stack without
// modifying the stack. It can be used to check if the stack is empty or
// to retrieve the top element without removing it.
func (s *Stack) PopRead() any {
	// Check if the stack is empty
	if e := s.list.Front(); e != nil {
		// If the stack is not empty, return the value of the top element
		return e.Value
	}
	// If the stack is empty, return nil
	return nil
}

// Pop removes and returns the element at the top of the stack.
// If the stack is empty, it returns nil.
//
// The element is removed from the stack and returned.
// If the stack is empty, it returns nil.
func (s *Stack) Pop() any {
	if e := s.list.Front(); e != nil {
		s.list.Remove(e)
		return e.Value
	}
	return nil
}

// PopAll removes all elements in the stack and returns them as a slice.
// The order of the elements in the slice is the same as the order of the
// elements in the stack, with the top element of the stack being the first
// element in the slice.
// If the stack is empty, it returns an empty slice.
func (s *Stack) PopAll() (res []any) {
	for s.list.Len() > 0 {
		res = append(res, s.Pop())
	}
	return res
}

// IsEmpty checks if the stack is empty.
//
// This method returns true if the stack contains no elements, and false
// otherwise. It is useful for determining whether any elements are present
// in the stack before attempting operations that require a non-empty stack.
func (s *Stack) IsEmpty() bool {
	return s.list.Len() == 0
}
