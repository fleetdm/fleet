# ADR-NNNN: Adopt use of error codes in Fleet API error responses

## Status

Proposed <!-- | Accepted | Rejected | Deprecated | Superseded -->
 
## Date

2026-01-29

## Context

When a Fleet API responds with an error, the response typically contains an `errors` array where each member has a `name` and a `reason`. The `reason` string may have been provided by a Fleet engineer when developing the error handling for the API, or it may have bubbled up from a low-level error (for example, a MySQL error). When presented with an API error response, front-end code currently uses several different tactics to negotiate the error, including:

* Examine the HTTP status code
* Match the `reason` of the first error against a substring (e.g. check if the `reason` includes the words `entity already exists`)
* Match the `name` of the first error against a string (e.g. check if the `name` is `install script`)

It then may choose to display a message to the end-user, again using several different methods, including:

* Displaying the `reason` verbatim
* Displaying another response property (e.g. `message`) verbatim
* Displaying a static string hard-coded in the front-end code, or a dynamic string constructed by front-end logic

The multiple ways that errors can be negotiated and displayed presents several challenges:

1. It is unclear whether the front-end or back-end is ultimately responsible for how the error is displayed to the end-user (if at all).
2. String-matching on the `reason` in the API response is brittle, especially when the `reason` is intended as a human-readable error explanation (as that kind of wording tends to change over time)
3. Making simple changes to error messages becomes cumbersome, especially if the error message is "load bearing" because front-end code matches against it. Changes can break front-end functionality, leading to the requirement of adding contract tests to APIs.
4. Localization of error messages is difficult because the messages are spread over both the front and back end.  
5. The error displayed to the end user may differ greatly depending on the context; i.e. a user calling the API directly may see a different message than a user in the UI.

## Decision

* Add a new `error_code` property to API errors which, when provided, becomes the source of truth for what the error is. 
* Always defer to using the `error_code` on the front-end when negotiating API errors (falling back to status code and other strategies when `error_code` is not present). 
* Maintain a mapping of `error_code` to human-readable error messages (or functions to retrieve the message, see "Error map implementation" below). If an API error response contains an `error_code` and that code has an entry in the map, use that text as the `reason` in the API response whenever possible.

### Requirements for `error_code` values

* `error_code` should be an all-caps, English-language text string of arbitrary length (but as short as possible while expressing the error state). Examples:
  * E_LABEL_NAME_CONFLICTS_BUILTIN
  * E_LABEL_NAME_EXISTS
  * E_LABEL_NOT_FOUND
  * E_POLICY_NOT_FOUND
  * E_POLICY_NAME_EXISTS
  * E_VALIDATION_FAILED
* The code should be unambiguous to the point where it can be used as the sole discriminator in error-handling logic. That is, it should allow the error handler to decide what to _do_ about the error without any further information.

### Error map implementation

The mapping of error codes to human-readable messages should be maintained on the backend. This will allow us to continue using the `reason` key in the API response to describe the error to the end user. The keys of the map should be error codes as described above, and the values should be either:

* Static strings intended to be displayed verbatim, e.g. "A label with this name already exists"
* String templates intended to be interpolated, e.g. "Cannot add label {label_name} because it conflicts with the name of a built-in label"
* Functions that, when called with the error key and additional data as arguments, return a static or template string. An example use case is for returning messages for validation errors such as "Name contains invalid characters" or "Name too long".

The error map should export constants to be used in code, e.g. `ErrCodelabelNameConflictsBuiltin = "E_LABEL_NAME_CONFLICTS_BUILTIN"`, so that service/module-layer code doesn't have to reference the static code strings directly.

### Using error codes in practice

_This is speculative, actual implementation to be determined by engineers after general strategy is approved._

We already have several helper methods for creating errors to return, such as `NewUserMessageError()` and `NewInvalidArgumentError()`; we could extend these to e.g. `NewUserMessageErrorWithCode()`, along with a generic `NewErrorWithCode()` helper. We'll also want a new interface like `ErrWithCode` that declares an `ErrorCode()` method, and in the [transport_error.go](https://github.com/fleetdm/fleet/blob/main/server/platform/endpointer/transport_error.go) code detect this to 1) add the `error_code` to the response and 2) derive the `reason` from the error code. For error codes that map to a string template or a function, we may need additional interface methods like `ErrorMetadata()`. 

### Where to start (and what to avoid)

* The first use of error codes should be to replace instances of `reasonIncludes` and `nameIncludes` in the front-end code. 
* We can experiment with adding new errors and replacing existing errors as we come across them to see how much of a pain the pattern is to use in practice.
* Errors that are bubbled up from lower-level methods can be left alone, or eventually share a generic `UNKNOWN_ERROR` code with no mapping (so that they continue to provide their own "reason").


## Consequences

Advantages of this approach:

- Front-end error negotitiation is simplified: just look at the error code to decide what to do.
- Front-end error messaging is simplified: in many cases, just use the "reason" verbatim.
- Anyone can easily update an error message just by updating the map: you just have to know the error code (which you can see in the API response), rather than having to find where in the code the error is returned.
- Error messages can be localized more easily since they're more centralized.

Drawbacks of this approach:

- Back-end service code that returns errors now has a layer between it and the error message. Looking at a line of code like
```
return nil, fleet.NewCodedError(ErrCodeLabelNameConflictsBuiltin, errMeta{ "Name": label.Name })
```
you can't tell at a glance what error message will be returned.
- For error map entries that use a function rather than a string value, updating the errors is slightly less straightforward than it was previously. However, this method should be used sparingly (mainly for validation errors), and implemented mainly as another map (of validation error "sub-codes" to strings).
- It's one more strategy for front-end devs to have to implement, albeit a better one. We likely won't fully switch over to this for some time, if ever.

## Alternatives considered

- Do nothing
  - Always the lowest-touch option! But doesn't solve any of our problems.

- Implement error codes, but only for error negotation -- not for providing human-readable error messages
  - Solves one problem (brittle error-handling code) but doesn't address the issues with finding and updating error strings in the app.

- Implement rerror codes + error map, but maintain the map in _front end_ code.
  - This is more in line with a "back end provides the data, front end visualizes it" mentality, which has a lot of value but is at odds with Fleet's API-first ethos. API-only users should still have helpful error messages.


## References

- [Back-end sync discussion notes](https://docs.google.com/document/d/1jRpQrpDw1Y9rBgoP6zveMfUlw2LuRRt_KH6DWlVyXt4/edit?tab=t.xd3ec8hrcez0#heading=h.yrt6iicxhpgs)
- [Stripe](https://docs.stripe.com/error-codes) uses error codes to help end-users negotiate errors
