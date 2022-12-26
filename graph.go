// Copyright (c) 2022, R.I. Pienaar and the Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package asyncjobs

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Item struct {
	value   any
	name    string
	edges   []*Item
	parents []*Item
}

type Graph struct {
	items map[string]*Item
	edges map[string][]*Item
	root  *Item
	mu    sync.Mutex
}

func NewItem(n string, v any) *Item {
	return &Item{name: n, value: v}
}

func NewGraph() *Graph {
	return &Graph{
		items: make(map[string]*Item),
		edges: make(map[string][]*Item),
	}
}

func (g *Graph) nodeNames() []string {
	names := make([]string, len(g.items))
	ctr := 0
	for n, _ := range g.items {
		names[ctr] = n
		ctr++
	}

	sort.Strings(names)

	return names
}

func (g *Graph) String() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	vals := make([]string, len(g.items))

	for i, name := range g.nodeNames() {
		var parts []string

		for j := 0; j < len(g.items[name].edges); j++ {
			parts = append(parts, g.items[name].edges[j].name)
		}
		sort.Strings(parts)

		vals[i] = fmt.Sprintf("%s -> %s", name, strings.Join(parts, ", "))
	}

	return strings.Join(vals, "\r")
}

func (g *Graph) AddItemRelation(i1 *Item, i2 *Item) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.edges[i1.name] = append(g.edges[i1.name], i2)
	i2.parents = append(i2.parents, i1)
	i1.edges = append(i1.edges, i2)

	if g.isCyclic(i1) {
		g.edges[i1.name] = g.edges[i1.name][0 : len(g.edges[i1.name])-1]
		i2.parents = i2.parents[0 : len(i2.parents)-1]
		i1.edges = i1.edges[0 : len(i1.edges)-1]
		return fmt.Errorf("cyclic")
	}

	return nil
}

func (g *Graph) isCyclic(item *Item) bool {
	if len(item.parents) == 0 {
		return false
	}

	stack := []*Item{item}
	visited := map[string]struct{}{}
	var current *Item

	for {
		if len(stack) == 0 {
			break
		}

		current, stack = stack[0], stack[1:]
		_, ok := visited[current.name]
		if ok {
			return true
		}
		visited[current.name] = struct{}{}

		stack = append(stack, current.parents...)
	}

	return false
}

func (g *Graph) WalkForExecution(cb func(*Item) error) error {
	if len(g.items) == 0 {
		return fmt.Errorf("no items")
	}

	stack := []*Item{}
	for _, i := range g.items {
		if len(i.parents) == 0 {
			stack = append(stack, i)
		}
	}

	wg := sync.WaitGroup{}

	visited := map[string]struct{}{}
	errors := make(chan error, len(g.items))
	mu := sync.Mutex{}

	visit := func(wg *sync.WaitGroup, i *Item) {
		defer wg.Done()
		err := cb(i)
		if err != nil {
			errors <- err
			return
		}

		mu.Lock()
		visited[i.name] = struct{}{}
		stack = append(stack, i.edges...)
		mu.Unlock()
	}

	for {
		if len(stack) == 0 {
			return nil
		}

		var current *Item
		mu.Lock()
		current, stack = stack[0], stack[1:]
		mu.Unlock()

		// if we dont yet have all parent relations satisfied we put it to the end
		for _, p := range current.parents {
			mu.Lock()
			_, ok := visited[p.name]
			mu.Unlock()

			if !ok {
				mu.Lock()
				stack = append(stack, current)
				mu.Unlock()

				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		wg.Add(1)
		go visit(&wg, current)

		mu.Lock()
		l := len(stack)
		mu.Unlock()
		if l == 0 {
			wg.Wait()
		}
	}
}

func (g *Graph) AddItem(item *Item) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	_, ok := g.items[item.name]
	if ok {
		return fmt.Errorf("item not unique")
	}

	g.items[item.name] = item

	if len(g.items) == 1 {
		g.root = item
	}

	return nil
}

func (g *Graph) GetItem(name string) (*Item, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	i, ok := g.items[name]

	return i, ok
}
