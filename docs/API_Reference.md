# REST API Reference

## Authentication

A Syndication server allows authentication operations over HTTP. However, these can be turned off by an administrator.  If the server does not allow an authentication operation a 403 Forbidden error.

### Register a new user

```
POST /register
```
#### Request

##### Parameters
|    Name    |  Type  |                Description              |
| ---------- | ------ | --------------------------------------- |
|  username  | string | **Required**. An alpha-numeric username |
|  password  | string | **Required**. A password                |
```bash
curl -d "username=foo" -d "password=pass"
```

#### Response

```
Status: 204 No Content
```

### Login a user

#### Request

```
POST /login
```

##### Parameters

|    Name    |  Type  |               Description               |
| ---------- | ------ | --------------------------------------- |
|  username  | string | **Required**. An alpha-numeric username |
|  password  | string | **Required**. A password                |

#### Response

```
Status: 200 OK

  {
    'token': 'Ad83...'
    'expiration': '2017-08-29'
  }
```

## Feeds

### Add a feed

```
POST /feeds
```

##### Parameters

| Name | Type | Description |
| ---- | ---- | ------------|
| title | string | A title to give to a subscribing feed. If this is not provided, the title found in the subscription will be used. |
| subscription | string | **Required.** A URL to a feed. This must point to a valid atom or rss feed. |

A `category` object can also be provided.

| Name | Type | Description |
| ---- | ---- | ------------|
|  id  | string | The id that the category should belong to. |

```
{
  'title' : 'Deeplinks',
  'subscription' : 'https://www.eff.org/rss/updates.xml',
  'category' :  {
    'id' : 'df10d51f-eb45-4f05-a20f-c18ae9f09b86'
  }
}
```

#### Response

```
Status: 201 Created
```
```
{
  'id' : 'e00aae3f-4c0d-403e-bb72-f3b99e20834a',
  'title' : 'Deeplinks',
  'author' ; 'Electronic Frontier Foundation',
  'description' : 'EFF's Deeplinks Blog: Noteworthy news from around the internet',
  'subscription' : 'https://www.eff.org/rss/updates.xml',
  'source' : 'http://eff.org',
  'status' : 'recheable',
  'category' :  {
    'name' : 'News',
    'id' : 'df10d51f-eb45-4f05-a20f-c18ae9f09b86'
  }
}
```

### Fetch a feed

```
GET /feeds/:feedID
```

#### Response
```
Status: 201 Created
```
```
{
  'id' : 'e00aae3f-4c0d-403e-bb72-f3b99e20834a',
  'title' : 'Deeplinks',
  'author' ; 'Electronic Frontier Foundation',
  'description' : 'EFF's Deeplinks Blog: Noteworthy news from around the internet',
  'subscription' : 'https://www.eff.org/rss/updates.xml',
  'source' : 'http://eff.org',
  'status' : 'recheable',
  'category' :  {
    'name' : 'News',
    'id' : 'df10d51f-eb45-4f05-a20f-c18ae9f09b86'
  }
}
```


### Get a list of feeds

```
GET /feeds
```

#### Response

```
Status: 200 OK
```
```
{
  'feeds': [
    {
      'id' : 'e00aae3f-4c0d-403e-bb72-f3b99e20834a',
      'title' : 'Deeplinks',
      'author' ; 'Electronic Frontier Foundation',
      'description' : 'EFF's Deeplinks Blog: Noteworthy news from around the internet',
      'subscription' : 'https://www.eff.org/rss/updates.xml',
      'source' : 'http://eff.org',
      'status' : 'recheable',
      'category' :  {
        'name' : 'News',
        'id' : 'df10d51f-eb45-4f05-a20f-c18ae9f09b86'
      }
    },
  ...
  ]
}
```

### Edit a feed

```
PUT /feeds/:feedID
```

```
{
  'title' : 'Deeplinks'
}
```

#### Response

```
Status: 204 No Content
```

### Delete a feed

```
DELETE /feeds/:feedID
```

#### Response
```
Status: 201 No Content
```

### Get entries from feed

```
GET /feeds/:feedID/entries
```

#### Request

##### Parameters

| Name | Type | Description |
| ---- | ---- | ----------- |
| markedAs | string | Return only entries marked as `read` or `unread` |
| pageSize | integer | Size of the returned page |
| page | integer | Page number for the returned entry list
| orderBy | string | Order entries by `newest` or `oldest`
| newerThan | integer | Return entries newer than a provided time in Unix format |

```
https://localhost:8081/v1/feeds/e00aae3f-4c0d-403e-bb72-f3b99e20834a/entries?markedAs=unread&pageSize=100&page=2&orderBy=newest&newerThan=1496116444
```

#### Response

```
{
  "entries" : [
    {
      'id' : 'cb7fac24-ec4a-4596-af89-19ad21d61e3e',
      'title' : 'A Bad Broadband Market Begs for Net Neutrality Protections',
      'description' : 'Anyone who has spent hours on...',
      'link' : 'https://www.eff.org/deeplinks/2017/05/bad-broadband-market-begs-net-neutrality-protections'
      'published' : '2017-05-30T03:26:38Z'
      'author' : 'Kate Tummarello',
      'isSaved' : 'true',
      'markedAs' : 'unread'
    },
    ...
  ]
}
```

### Mark a feed

```
PUT /feeds/:feedID/mark
```

#### Parameters

| Name | Type | Description |
| ---- | ---- | ----------- |
| as   | string | The marker to apply to the feed. This can be either `read` or `unread`

```
http://locahost:8080/feeds/e00aae3f-4c0d-403e-bb72-f3b99e20834a/mark?as=read
```

### Get stats for a feed

```
GET /feeds/:feedID/stats
```

#### Response
```
{
  'unread' : 48
  'read' : 123
  'saved' : 23
  'total' : 171
}
```

## Entries

### Get Entry

```
GET /entries/:entryID
```

#### Response

```
{
  'id' : 'cb7fac24-ec4a-4596-af89-19ad21d61e3e',
  'title' : 'A Bad Broadband Market Begs for Net Neutrality Protections',
  'description' : 'Anyone who has spent hours on...',
  'link' : 'https://www.eff.org/deeplinks/2017/05/bad-broadband-market-begs-net-neutrality-protections'
  'published' : '2017-05-30T03:26:38Z'
  'author' : 'Kate Tummarello',
  'isSaved' : 'true',
  'markedAs' : 'unread'
}
```

### Get Entries

```
GET /entries
```

| Name | Type | Description |
| ---- | ---- | ----------- |
| markedAs | string | Return only entries marked as `read` or `unread` |
| pageSize | integer | Size of the returned page |
| page | integer | Page number for the returned entry list
| orderBy | string | Order entries by `newest` or `oldest`
| newerThan | integer | Return entries newer than a provided time in Unix format |

```
https://localhost:8081/v1/entries?markedAs=unread&pageSize=100&page=2&orderBy=newest&newerThan=1496116444
```

### Mark entry

```
PUT /entries/:entryID/mark
```

#### Parameters

| Name | Type | Description |
| ---- | ---- | ----------- |
| as   | string | The marker to apply to the feed. This can be either `read` or `unread`

```
http://locahost:8080/entries/cb7fac24-ec4a-4596-af89-19ad21d61e3e/mark?as=read
```

### Get stats for entries

```
GET /entries/stats
```

#### Response

```
{
  'unread' : 48
  'read' : 123
  'saved' : 23
  'total' : 171
}
```

## Categories

### Create a Category

```
POST /categories
```

#### Request

##### Parameters

```
{
  'name': 'News'
}
```

#### Response
```
Status: 201 Created
```

```
{
  'id': '84a9497e-d165-4fb9-a48e-be85bc9ff559',
  'name': 'News'
}
```

### Get categories

```
GET /categories
```

#### Response

```
{
  'categories': [
    {
      'id': '84a9497e-d165-4fb9-a48e-be85bc9ff559',
      'name': 'News'
    },
    ...
  ]
}
```

### Get a category

```
GET /categories/:categoryID
```

#### Response
```
Status: 200 OK
```

```
{
  'id': '84a9497e-d165-4fb9-a48e-be85bc9ff559',
  'name': 'News'
}
```

### Edit a category

```
PUT /categories/:categoryID
```

#### Request

```
{
  'name': 'Activism'
}
```

#### Response

```
Status: 204 OK
```

### Delete a category

```
DELETE /categories/:categoryID
```

#### Response

```
Status: 201 No Content
```

### Get feeds

```
GET /categories/:categoryID/feeds
```

#### Response

```
{
  'feeds' : [
    {
      'id' : 'e00aae3f-4c0d-403e-bb72-f3b99e20834a',
      'title' : 'Deeplinks',
      'author' ; 'Electronic Frontier Foundation',
      'description' : 'EFF's Deeplinks Blog: Noteworthy news from around the internet',
      'subscription' : 'https://www.eff.org/rss/updates.xml',
      'source' : 'http://eff.org',
      'status' : 'recheable'
    },
    ...
  ]
}
```

### Add feed to a category

```
PUT /categories/:categoryID/feeds
```

#### Request

##### Parameters

| Name | Type | Description |
| ---- | ---- | ----------- |
| feeds | `array` of `string`s | A list of feed IDs that will be added to the category

```
{
  'feeds' : [
    '6134d5cc-b595-4263-aea0-53900d9d4ae8',
    '59652914-fd89-4f1d-9a50-6437fb0aa3c8',
    'bca982b7-5c26-4da9-b565-d5ab0fcc487c',
    ...
  ]
}
```

#### Response
```
Status: 202 Accepted
```

### Get entries from a category

```
GET /categories/:categoryID/entries
```
##### Parameters

| Name | Type | Description |
| ---- | ---- | ----------- |
| markedAs | string | Return only entries marked as `read` or `unread` |
| pageSize | integer | Size of the returned page |
| page | integer | Page number for the returned entry list
| orderBy | string | Order entries by `newest` or `oldest`
| newerThan | integer | Return entries newer than a provided time in Unix format |

```
https://localhost:8081/v1/categories/84a9497e-d165-4fb9-a48e-be85bc9ff559/entries?markedAs=unread&pageSize=100&page=2&orderBy=newest&newerThan=1496116444
```

#### Response
```
Status: 200 OK
```

```
{
  "entries" : [
    {
      'id' : 'cb7fac24-ec4a-4596-af89-19ad21d61e3e',
      'title' : 'A Bad Broadband Market Begs for Net Neutrality Protections',
      'description' : 'Anyone who has spent hours on...',
      'link' : 'https://www.eff.org/deeplinks/2017/05/bad-broadband-market-begs-net-neutrality-protections'
      'published' : '2017-05-30T03:26:38Z'
      'author' : 'Kate Tummarello',
      'isSaved' : 'true',
      'markedAs' : 'unread'
    },
    ...
  ]
}
```
