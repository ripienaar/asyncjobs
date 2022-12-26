// Copyright (c) 2022, R.I. Pienaar and the Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package asyncjobs

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Graph", func() {
	Describe("Basics", func() {
		It("Should function", func() {
			graph := NewGraph()

			var items []*Item

			for i := 0; i < 10; i++ {
				items = append(items, NewItem(fmt.Sprintf("item %d", i), struct{}{}))
				graph.AddItem(items[i])
			}

			Expect(graph.AddItemRelation(items[1], items[2])).ToNot(HaveOccurred())
			Expect(graph.AddItemRelation(items[1], items[3])).ToNot(HaveOccurred())
			Expect(graph.AddItemRelation(items[1], items[4])).ToNot(HaveOccurred())
			Expect(graph.AddItemRelation(items[4], items[5])).ToNot(HaveOccurred())
			Expect(graph.AddItemRelation(items[4], items[1])).To(MatchError("cyclic"))
			Expect(graph.AddItemRelation(items[4], items[4])).To(MatchError("cyclic"))
			Expect(items[4].edges).To(HaveLen(1))
			Expect(items[4].edges[0].name).To(Equal("item 5"))

			Expect(graph.items).To(HaveLen(10))
			Expect(graph.items["item 1"].edges).To(Equal([]*Item{items[2], items[3], items[4]}))
			Expect(graph.items["item 5"].parents).To(Equal([]*Item{items[4]}))
			Expect(graph.items["item 0"].edges).To(HaveLen(0))
			Expect(graph.items["item 0"].parents).To(HaveLen(0))

			graph.WalkForExecution(func(i *Item) error {
				fmt.Println(i.name)
				return nil
			})
		})
	})
})
