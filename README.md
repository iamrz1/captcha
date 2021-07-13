# captcha
## Generate:

- endpoint: `/api/v1/get-captcha`
- method: GET
- request: nil
- response:
```json
{
  "data": {
    "id":"g8KasGg9orTv83X0qfcv",
    "captcha":"data:image/png;base64,iVwszsCDgNmB3SIb0W...gtZ=="
  },
  "message":"Captcha generated",
  "success":true
}
```

## Verify:

- endpoint: `/api/v1/verify-captcha`
- method: POST
- request:
```json
{
  "id":"g8KasGg9orTv83X0qfcv",
  "value":"12345"

}
```
- response:
```json
{
  "message":"Captcha verified",
  "success":true
}
```