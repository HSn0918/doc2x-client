# doc2x-client

Go client for doc2x API v2 - PDF parsing and document conversion.

**Documentation:** https://noedgeai.feishu.cn/wiki/Q8QIw3PT7i4QghkhPoecsmSCnG1

## Installation

```bash
go get github.com/hsn0918/doc2x-client
```

## Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/hsn0918/doc2x-client"
)

func main() {
    // Create client
    c := client.NewClient(client.WithAPIKey("your-api-key"))

    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    // Upload PDF
    resp, err := c.UploadPDF(ctx, pdfData)
    if err != nil {
        log.Fatal(err)
    }

    // Wait for parsing
    status, err := c.WaitForParsing(ctx, resp.Data.UID, 3*time.Second)
    if err != nil {
        log.Fatal(err)
    }

    // Access parsed markdown
    for _, page := range status.Data.Result.Pages {
        println(page.Md)
    }
}
```

## License

MIT