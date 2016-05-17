package main

import (
	"fmt"
	"strconv"
)

// Pagination Component for html

type PageItem struct {
	Label     string
	Url       string
	IsCurrent bool
}

func (this *PageItem) Class() string {
	if this.IsCurrent {
		return "active"
	}
	return ""
}

type Pager struct {
	CurrentPage int
	PageSize    int
	Total       int
	MaxItem     int
	UrlPattern  string
}

func NewPager(pageSize int, total int) *Pager {
	return &Pager{0, pageSize, total, 10, ""}
}

func (this *Pager) Offset() int {
	return this.CurrentPage * this.PageSize
}

func (this *Pager) Limit() int {
	return this.PageSize
}

func (this *Pager) Page() int {
	return (this.Total + this.PageSize - 1) / this.PageSize
}

func (this *Pager) IsVisible() bool {
	return this.Total > this.PageSize
}

func (this *Pager) IsFirstVisible() bool {
	startPage := MaxInt(0, this.CurrentPage-this.MaxItem/2)
	return startPage > 0
}

func (this *Pager) FirstItem() *PageItem {
	url := fmt.Sprintf(this.UrlPattern, this.PageSize, 0)
	return &PageItem{"Start", url, false}
}

func (this *Pager) IsEndVisible() bool {
	startPage := MaxInt(0, this.CurrentPage-this.MaxItem/2)
	return startPage+this.MaxItem < this.Page()
}

func (this *Pager) EndItem() *PageItem {
	url := fmt.Sprintf(this.UrlPattern, this.PageSize-1, this.Page())
	return &PageItem{"End", url, false}
}

func (this *Pager) Pages() []*PageItem {
	startPage := MaxInt(0, this.CurrentPage-this.MaxItem/2)
	maxPage := this.Page()
	pages := []*PageItem{}
	for i := startPage; i < startPage+this.MaxItem && i < maxPage; i++ {
		isCurrent := i == this.CurrentPage
		url := fmt.Sprintf(this.UrlPattern, i)
		item := &PageItem{strconv.Itoa(i + 1), url, isCurrent}
		pages = append(pages, item)
	}
	return pages
}
