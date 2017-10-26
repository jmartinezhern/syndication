# REST API Reference

## Authentication

A Syndication server allows authentication operations over HTTP. However, these can be turned off by an administrator.  If the server does not allow an authentication operation a 403 Forbidden error will be returned.

### Register a new user

```
POST /register
```

#### Parameters

|    Name    |  Type  |                  Description                |
| ---------- | ------ | ------------------------------------------- |
|  username  | string | **Required**. An alpha-numeric username.    |
|  password  | string | **Required**. A password.                   |

```bash
curl -d "username=foo" -d "password=pass" http://localhost:8080/v1/login
```

#### Response

```
Status: 204 No Content
```

### Login a user

```
POST /login
```

#### Parameters

|    Name    |  Type  |               Description               |
| ---------- | ------ | --------------------------------------- |
|  username  | string | **Required**. An alpha-numeric username |
|  password  | string | **Required**. A password                |

#### Response

```
Status: 200 OK
```
```javascript
  {
    'token': 'Ad83...'
    'expiration': '2017-08-29'
  }
```

## Entries

### Get an Entry's information

```
GET /entries/:entryID
```

#### Response

```
Status: 200 OK
```

```javascript
{
  'id' : 'MTUwNDgwNTA3Nw==',
  'title' : 'A Bad Broadband Market Begs for Net Neutrality Protections',
  'description' : 'Anyone who has spent hours on...',
  'link' : 'https://www.eff.org/deeplinks/2017/05/bad-broadband-market-begs-net-neutrality-protections'
  'published' : '2017-05-30T03:26:38Z'
  'author' : 'Kate Tummarello',
  'isSaved' : 'true',
  'markedAs' : 'unread'
}
```

### Get a list of all Entries

```
GET /entries
```

|    Name   |   Type  |                               Description                               |
| --------- | ------- | ----------------------------------------------------------------------- |
| markedAs  | string  | Return only entries marked as `read` or `unread`. Default is `unread`.  |
| pageSize  | integer | Size of the returned page. Default is 100.                              |
| page      | integer | Page number for the returned entry list. Default is 1.                  |
| orderBy   | string  | Order entries by `newest` or `oldest`. Default is `newest`.             |
| newerThan | integer | Return entries newer than a provided time in Unix format.               |

```bash
curl -H "Authorization: Bearer A32wdj48..." https://localhost:8081/v1/entries?markedAs=unread&pageSize=100&page=2&orderBy=newest&newerThan=1496116444
```
#### Response

```
Status: 200 OK
```

```javascript
{
  "entries" : [
    {
      'id' : 'MTUwNDgwNTA3Nw==',
      'title' : 'A Bad Broadband Market Begs for Net Neutrality Protections',
      'description' : 'Anyone who has spent hours on...',
      'link' : 'https://www.eff.org/deeplinks/2017/05/bad-broadband-market-begs-net-neutrality-protections'
      'published' : '2017-05-30T03:26:38Z'
      'author' : 'Kate Tummarello',
      'isSaved' : true,
      'markedAs' : 'unread'
    },
    ...
  ]
}
```

### Apply a Marker to an Entry

```
PUT /entries/:entryID/mark
```

#### Parameters

| Name | Type | Description |
| ---- | ---- | ----------- |
| as   | string | The marker to apply to the feed. This can be either `read` or `unread`

```bash
curl -X PUT -H "Authorization: Bearer Adk4maY..." http://locahost:8080/entries/MTUwNDgwNTA3Nw==/mark?as=read
```
#### Response
```
Status: 204 No Content
```

### Get stats for all Entries

```
GET /entries/stats
```

#### Response

```
Status: 200 OK
```

```javascript
{
  'unread' : 48
  'read' : 123
  'saved' : 23
  'total' : 171
}
```

## Feeds

### Subscribe to a feed

```
POST /feeds
```

#### Parameters

| Name  |  Type  | Description |
| ----  | ------ | ------------|
| title | string | Title for the subscribed feed. If one is not provided, the title found in the subscription will be used. |
| subscription | string | **Required.** URL to a feed. This must point to a valid Atom or RSS feed. |

A `category` object can also be provided.

| Name | Type | Description |
| ---- | ---- | ------------|
|  id  | string | The id that the category should belong to. |

```javascript
{
  'title' : 'Deeplinks',
  'subscription' : 'https://www.eff.org/rss/updates.xml',
  'category' :  {
    'id' : 'MTUwNDgwNDQ4Nw=='
  }
}
```

#### Response

```
Status: 201 Created
```
```javascript
{
  'id' : 'MTUwNDgwNDQ4Nw==',
  'title' : 'Deeplinks',
  'author' ; 'Electronic Frontier Foundation',
  'description' : 'EFFs Deeplinks Blog: Noteworthy news from around the internet',
  'subscription' : 'https://www.eff.org/rss/updates.xml',
  'source' : 'http://eff.org',
  'status' : 'reachable',
  'category' :  {
    'name' : 'News',
    'id' : 'MTUwNDgwNDU5Mg=='
  }
}
```

### Get a Feed's metadata

```
GET /feeds/:feedID
```

#### Response
```
Status: 201 Created
```
```javascript
{
  'id' : 'MTUwNDgwNDU5Mg==',
  'title' : 'Deeplinks',
  'author' ; 'Electronic Frontier Foundation',
  'description' : 'EFFs Deeplinks Blog: Noteworthy news from around the internet',
  'subscription' : 'https://www.eff.org/rss/updates.xml',
  'source' : 'http://eff.org',
  'status' : 'reachable',
  'category' :  {
    'name' : 'News',
    'id' : 'MTUwNDgwNDQ4Nw=='
  }
}
```


### Get a list of subscribed Feeds

```
GET /feeds
```

#### Response

```
Status: 200 OK
```
```javascript
{
  'feeds': [
    {
      'id' : 'MTUwNDgwNDQ4Nw==',
      'title' : 'Deeplinks',
      'author' ; 'Electronic Frontier Foundation',
      'description' : 'EFFs Deeplinks Blog: Noteworthy news from around the internet',
      'subscription' : 'https://www.eff.org/rss/updates.xml',
      'source' : 'http://eff.org',
      'status' : 'recheable',
      'category' :  {
        'name' : 'News',
        'id' : 'MTUwNDgwNDU5Mg=='
      }
    }
    ...
  ]
}
```

### Edit a Feed's information

```
PUT /feeds/:feedID
```
```javascript
{
  'title' : 'Deeplinks'
}
```

#### Response

```
Status: 204 No Content
```

### Unsubscribe from Feed

```
DELETE /feeds/:feedID
```

#### Response
```
Status: 201 No Content
```

### Get a list of Entries from a Feed

```
GET /feeds/:feedID/entries
```

#### Parameters

|    Name   |   Type  |                               Description                               |
| --------- | ------- | ----------------------------------------------------------------------- |
| markedAs  | string  | Return only entries marked as `read` or `unread`. Default is `unread`.  |
| pageSize  | integer | Size of the returned page. Default is 100.                              |
| page      | integer | Page number for the returned entry list. Default is 1.                  |
| orderBy   | string  | Order entries by `newest` or `oldest`. Default is `newest`.             |
| newerThan | integer | Return entries newer than a provided time in Unix format.               |

```bash
curl -H "Authorization: Bearer Adae8kd..." https://localhost:8080/v1/feeds/MTUwNDgwNDQ4Nw==/entries?markedAs=unread&pageSize=100&page=2&orderBy=newest&newerThan=1496116444
```

#### Response

```
Status: 200 OK
```

```javascript
{
  "entries" : [
    {
      'id' : 'MTUwNDgwNTA3Nw==',
      'title' : 'A Bad Broadband Market Begs for Net Neutrality Protections',
      'description' : 'Anyone who has spent hours on...',
      'link' : 'https://www.eff.org/deeplinks/2017/05/bad-broadband-market-begs-net-neutrality-protections'
      'published' : '2017-05-30T03:26:38Z'
      'author' : 'Kate Tummarello',
      'isSaved' : true,
      'markedAs' : 'unread'
    },
    ...
  ]
}
```

### Apply a Marker to a Feed

```
PUT /feeds/:feedID/mark
```

#### Parameters

| Name |  Type  | Description |
| ---- | ----   | ----------- |
| as   | string | **Required**. The marker to apply to the feed. This can be either `read` or `unread` |

```bash
curl -X PUT -H "Authorization: Bearer Adj48dkx.." http://locahost:8080/feeds/MTUwNDgwNTA3Nw==/mark?as=read
```

#### Response
```
Status: 204 No Content
```

### Get stats for a Feed

```
GET /feeds/:feedID/stats
```

#### Response

```
Status: 200 OK
```

```javascript
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

#### Parameters

```javascript
{
  'name': 'News'
}
```

#### Response
```
Status: 201 Created
```

```javascript
{
  'id': 'MTUwNDgwNTA3Nw==',
  'name': 'News'
}
```

### Get a list of Categories

```
GET /categories
```

#### Response

```
Status: 200 OK
```

```javascript
{
  'categories': [
    {
      'id': 'MTUwNDgwNTA3Nw==',
      'name': 'News'
    },
    ...
  ]
}
```

### Get a Category's metadata

```
GET /categories/:categoryID
```

#### Response
```
Status: 200 OK
```

```javascript
{
  'id': 'MTUwNDgwNTA3Nw==',
  'name': 'News'
}
```

### Edit a Category's information

```
PUT /categories/:categoryID
```

```javascript
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

### Get a list of Feeds from a Category

```
GET /categories/:categoryID/feeds
```

#### Response

```
Status: 200 OK
```

```javascript
{
  'feeds' : [
    {
      'id' : 'MTUwNDgwNTA3Nw==',
      'title' : 'Deeplinks',
      'author' ; 'Electronic Frontier Foundation',
      'description' : 'EFFs Deeplinks Blog: Noteworthy news from around the internet',
      'subscription' : 'https://www.eff.org/rss/updates.xml',
      'source' : 'http://eff.org',
      'status' : 'reachable'
    },
    ...
  ]
}
```

### Add Feeds to a Category

```
PUT /categories/:categoryID/feeds
```

#### Parameters

| Name  |           Type       | Description |
| ----- | -------------------- | ----------- |
| feeds | `array` of `string`s | A list of IDs for Feeds that will be added to the category. |

```javascript
{
  'feeds' : [
    'MTUwNDgwNDQ4Nw==',
    'MTUwNDgwNDU5Mg==',
    'MTUwNDgwNTA3Nw==',
    ...
  ]
}
```

#### Response
```
Status: 201 No Content
```

### Get a list of Entries from a Category

```
GET /categories/:categoryID/entries
```
#### Parameters

|    Name   |   Type  |                               Description                               |
| --------- | ------- | ----------------------------------------------------------------------- |
| markedAs  | string  | Return only entries marked as `read` or `unread`. Default is `unread`.  |
| pageSize  | integer | Size of the returned page. Default is 100.                              |
| page      | integer | Page number for the returned entry list. Default is 1.                  |
| orderBy   | string  | Order entries by `newest` or `oldest`. Default is `newest`.             |
| newerThan | integer | Return entries newer than a provided time in Unix format.               |

```bash
curl -H "Authorization: Bearer Adae8kd..." https://localhost:8080/v1/categories/MTUwNDgwNDQ4Nw==/entries?markedAs=unread&pageSize=100&page=2&orderBy=newest&newerThan=1496116444
```

#### Response
```
Status: 200 OK
```

```javascript
{
  "entries" : [
    {
      'id' : 'MTUwNDgwNTA3Nw==',
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
### Get stats for a Category

```
GET /categories/:categoryID/stats
```

#### Response
```
Status: 200 OK
```

```javascript
{
  'unread' : 48
  'read' : 123
  'saved' : 23
  'total' : 171
}
```

### Apply a Marker to a Category

```
PUT /categories/:categoryID/mark
```

#### Parameters

| Name |  Type  | Description |
| ---- | ----   | ----------- |
| as   | string | **Required**. The marker to apply to the feed. This can be either `read` or `unread` |

```bash
curl -X PUT -H "Authorization: Bearer Adj48dkx.." http://locahost:8080/categories/MTUwNDgwNTA3Nw==/mark?as=read
```

#### Response
```
Status: 204 No Content
```

## Tags

### Create a tag

```
POST /tags
```

#### Parameters

```javascript
{
  'name': 'News'
}
```
#### Response
```
Status: 200 OK
```

### Get a list of tags

```
GET /tags
```

#### Response

```
Status: 200 OK
```

```javascript
{
  tags: [
    {
      'name': 'World News',
      'id': 'MTUwNDgwNTA3Nw=='
    },
    ...
  ]
}
```

### Get a tag's metadata

```
GET /tags/:tagID
```

#### Response

```
Status: 200 OK
```

```javascript
{
  'name': 'World News',
  'id': 'MTUwNDgwNTA3Nw=='
}
```

### Edit a tag

```
PUT /tags/:tagID
```

#### Parameters

``` javascript
{
  'name': 'International News'
}
```

#### Response

```
Status: 204 No Content
```

### Delete a tag

```
DELETE /tags/:tagID
```

#### Response

```
Status: 204 No Content
```


### Get a list of all Entries with a tag

```
GET /tags/:tagID/entries
```

|    Name   |   Type  |                               Description                               |
| --------- | ------- | ----------------------------------------------------------------------- |
| markedAs  | string  | Return only entries marked as `read` or `unread`. Default is `unread`.  |
| pageSize  | integer | Size of the returned page. Default is 100.                              |
| page      | integer | Page number for the returned entry list. Default is 1.                  |
| orderBy   | string  | Order entries by `newest` or `oldest`. Default is `newest`.             |
| newerThan | integer | Return entries newer than a provided time in Unix format.               |

```bash
curl -H "Authorization: Bearer A32wdj48..." https://localhost:8081/v1/tags/MTUwNDgwNTA3Nw==/entries?markedAs=unread&pageSize=100&page=2&orderBy=newest&newerThan=1496116444
```
#### Response

```
Status: 200 OK
```

```javascript
{
  "entries" : [
    {
      'id' : 'MTUwNDgwNTA3Nw==',
      'title' : 'A Bad Broadband Market Begs for Net Neutrality Protections',
      'description' : 'Anyone who has spent hours on...',
      'link' : 'https://www.eff.org/deeplinks/2017/05/bad-broadband-market-begs-net-neutrality-protections'
      'published' : '2017-05-30T03:26:38Z'
      'author' : 'Kate Tummarello',
      'isSaved' : true,
      'markedAs' : 'unread'
    },
    ...
  ]
}
```

### Apply a tag to an entry

```
PUT /tags/:tagID/entries
```

#### Parameters

```javascript
{
  "entries": [
    "MTUwNDgwNTA3Nw==",
    "MTUwNDgwNDQ4Nw==",
    ...
  ]
}
```

#### Response

```
Status: 204 No Content
```
