package common

import "container/list"

// Stack 定义了一个使用 list.List 实现的栈
type Stack struct {
	list *list.List
}

// NewStack 创建一个新的空栈
func NewStack() *Stack {
	return &Stack{
		list: list.New(),
	}
}

// Push 将一个元素压入栈顶
func (s *Stack) Push(v any) {
	s.list.PushFront(v)
}

// PopRead 读取但不移除栈顶元素
func (s *Stack) PopRead() any {
	if e := s.list.Front(); e != nil {
		return e.Value
	}
	return nil
}

// Pop 移除并返回栈顶元素
func (s *Stack) Pop() any {
	if e := s.list.Front(); e != nil {
		s.list.Remove(e)
		return e.Value
	}
	return nil
}

// PopAll 移除并返回所有栈顶元素
func (s *Stack) PopAll() (res []any) {
	for s.list.Len() > 0 {
		res = append(res, s.Pop())
	}
	return res
}

// IsEmpty 检查栈是否为空
func (s *Stack) IsEmpty() bool {
	return s.list.Len() == 0
}
