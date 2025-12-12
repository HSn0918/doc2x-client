# doc2x-client

面向 Doc2X API v2 的 Go SDK，覆盖 PDF 解析、导出、图片 layout OCR 等核心接口。

- 统一封装 `/parse/preupload`、`/parse/status`、`/convert/parse`、`/convert/parse/result`
- 内建轮询工具：`WaitForParsing` / `WaitForConversion` / `WaitForImageLayout`
- 同步/异步图片解析接口、OSS 直传辅助方法、结果下载工具

## 安装

```bash
go get github.com/hsn0918/doc2x-client
```

## 快速上手

```go
c := client.NewClient("sk-xxx")
ctx := context.Background()

pre, _ := c.PreUpload(ctx)
_ = c.UploadToPresignedURL(ctx, pre.Data.URL, pdfBytes)
status, _ := c.WaitForParsing(ctx, pre.Data.UID, 2*time.Second)

convertReq := client.ConvertRequest{UID: pre.Data.UID, To: "md", FormulaMode: "normal"}
_, _ = c.ConvertParse(ctx, convertReq)
result, _ := c.WaitForConversion(ctx, pre.Data.UID, 2*time.Second)
data, _ := c.DownloadFile(ctx, result.Data.URL)
```

图片 layout（≤7 MB）：

```go
layout, _ := c.ParseImageLayout(ctx, imageBytes) // 同步
job, _ := c.AsyncParseImageLayout(ctx, imageBytes) // 异步
layoutStatus, _ := c.WaitForImageLayout(ctx, job.Data.UID, 2*time.Second)
```

## 注意事项

- Base URL：`https://v2.doc2x.noedgeai.com`，务必直连；鉴权头 `Authorization: Bearer sk-xxx`
- 建议优先使用 `PreUpload + OSS PUT`，单次可达 1 GB；`UploadPDF` 仅适合 ≤300 MB
- 轮询频率 1–3 s，接口结果 24 h 过期；若遇到 `parse_*` 错误码请参考官方 FAQ

## 参考

- Doc2X API v2 官方文档：https://doc2x.noedgeai.com/help/zh-cn/api/doc2x-api-interface.html
- Doc2X 常见问题 / 图片接口文档：见飞书文档入口
