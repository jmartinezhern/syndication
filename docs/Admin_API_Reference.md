# Admin API Reference

## Basics

### Requests and Responses

All requests should be sent as JSON and should be sent to the configured Unix socket.

All responses have the following format:

```
{
  "status": 0,
  "error": "OK",
  "result": nil
}
```

Any result from an executed command will be returned in the result field.

The status code can be one of the following:

| Code | Meaning |
| ---- | ------- |
|  0   | **OK**. The action was successful. |
|  1   | **Not Implemented**. The command given is not implemented by the server. |
|  2   | **Unknown Command**. The command given is not understood by the server. |
|  3   | **Bad Request**. The request is malformed. |
|  4   | **Bad argument**. One of the given arguments is malformed. |
|  5   | **Database Error**. A database error occurred while the command was performed. |
|  6   | **Internal Error**. An error occurred while the command was performed. |


## Commands

### Create a user

#### Request

```
{
  "command": "NewUser",
  "arguments": {
    "username: "user",
    "password": "pasw"
  }
}
```

### Get a list of users

#### Request
```
{
 "command": "GetUsers"
}
```

#### Response

```
{
  "status": 0,
  "error": "OK",
  "result": [
    {
      "id": "MTUwNDgwNTA3Nw==",
      "username": "Gopher"
    },
    ...
  ]
}
```

### Get a user's information

#### Request

```
{
  "command": "GetUser",
  "arguments": {
    "userID": "MTUwNDgwNTA3Nw=="
  }
}
```

#### Response

```
{
  "status": 0,
  "error": "OK",
  "result": {
    "id": "MTUwNDgwNTA3Nw==",
    "username": "Gopher"
  }
}
```

### Delete a user

#### Request

```
{
  "command": "DeleteUser",
  "arguments": {
    "userID": "MTUwNDgwNTA3Nw=="
  }
}
```

### Change a user's name

#### Request

```
{
  "command": "ChangeUserName",
  "arguments": {
    "userID": "MTUwNDgwNTA3Nw==",
    "newName": "AwesomeGopher"
  }
}
```

### Change a user's password

#### Request

```
{
  "command": "ChangeUserPassword",
  "arguments": {
    "userID": "MTUwNDgwNTA3Nw==",
    "newPassword": "supersecure"
  }
}
```
