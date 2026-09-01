package response

// OffsetPageResp 通用offset分页响应模板，泛型
type OffsetPageResp[T any] struct {
	List      []T   `json:"list"`
	Total     int64 `json:"total"`
	Page      int   `json:"page"`
	PageSize  int   `json:"page_size"`
	TotalPage int64 `json:"total_page"`
}
