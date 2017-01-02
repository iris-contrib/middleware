# This is a middleware ported to Iris

## Link: [https://github.com/geekypanda/oauth2server](https://github.com/geekypanda/oauth2server)

## Documentation below

# OAuth 2.0 Authorization Server Middleware
OAuth 2.0 Authorization Server &amp; Authorization Middleware for the [Iris Web Framework](https://github.com/kataras/iris).

This library offers an OAuth 2.0 Authorization Server based on Iris and an Authorization Middleware usable in Resource Servers developed with Iris.

## Authorization Server
The Authorization Server is implemented by the struct _OAuthBearerServer_ that manages two grant types of authorizations (password and client_credentials).
This Authorization Server is made to provide authorization token usable for consuming resources API.

### Password grant type
_OAuthBearerServer_ supports password grant type allowing the token generation for username / password credentials.

### Client Credentials grant type
_OAuthBearerServer_ supports client_credentials grant type allowing the token generation for client_id / client_secret credentials.

### Authorization Code and Implicit grant type
These grant types are currently not supported.

### Refresh token grant type
If authorization token will expire, client can regenerate the token calling the authorization server and using the refresh_token grant type.

## Authorization Middleware
The Iris middleware _BearerAuthentication_ intercepts the resource server calls and authorizes only resource requests containing a valid bearer token.

## Token Formatter
Authorization Server crypts the token using the Token Formatter and Authorization Middleware decrypts the token using the same Token Formatter.
This library contains a default implementation of the formatter interface called _SHA256RC4TokenSecureFormatter_ based on the algorithms SHA256 and RC4.
Programmers can develop their Token Formatter implementing the interface _TokenSecureFormatter_ and this is really recommended before publishing the API in a production environment.

## Credentials Verifier
The interface _CredentialsVerifier_ defines the hooks called during the token generation process.
The methods are called in this order:
- _ValidateUser() or ValidateClient()_ called first for credentials verification
- _AddClaims()_ used for add information to the token that will be encrypted
- _StoreTokenId()_ called after the token generation but before the response, programmers can use this method for storing the generated Ids
- _AddProperties()_ used for add clear information to the response

There is another method in the _CredentialsVerifier_ interface that is involved during the refresh token process.
In this case the methods are called in this order:
- _ValidateTokenId()_ called first for TokenId verification, the method receives the TokenId related to the token associated to the refresh token
- _AddClaims()_ used for add information to the token that will be encrypted
- _StoreTokenId()_ called after the token regeneration but before the response, programmers can use this method for storing the generated Ids
- _AddProperties()_ used for add clear information to the response

## Authorization Server usage example
This snippet shows how to create an authorization server
```go
package main

import (
	"time"

	"github.com/geekypanda/oauth2server"
	"github.com/kataras/iris"
)

func main() {
  s := oauth2server.NewOAuthBearerServer(
		"mySecretKey-10101",
		time.Second*120,
		&TestUserVerifier{},
		nil)
	iris.Post("/token", s.UserCredentials)
	iris.Post("/auth", s.ClientCredentials)

	iris.Listen(":9090")
}
```

## Authorization Middleware usage example
This snippet shows how to use the middleware
```go
    authorized := iris.Party("/authorized")
	// use the Bearer Athentication middleware
	authorized.Use(oauth2server.Authorize("mySecretKey-10101", nil))

	authorized.Get("/customers", GetCustomers)
	authorized.Get("/customers/:id/orders", GetOrders)
```

> Note that the authorization server and the authorization middleware are both using the same token formatter and the same secret key for encryption/decryption.

## Reference
- [OAuth 2.0 RFC](https://tools.ietf.org/html/rfc6749)
- [OAuth 2.0 Bearer Token Usage RFC](https://tools.ietf.org/html/rfc6750)

## License
[MIT](https://github.com/maxzerbini/oauth/blob/master/LICENSE)
