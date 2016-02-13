**See detail of logged in user**
----
  See detail of a user. This endpoint is accessible to a logged in user for their own username regardless of permissions. For details of user not currently logged in use /users endpoint.

* **URL**

  /user

* **Permission**

  None

* **Method:**

  `GET`
  
* **URL Params**
  
  **Required:**

  `AuthToken[string]`

* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [], Username: "haisum", Email: "", Permissions: [...],  Request: {..}} }`
 
* **Error Response:**

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {...} }`

  OR

  * **Code:** 401 UNAUTHORIZED <br />
    **Content:** `{ Errors : ["You're not authorized to perform this action."], Request : {...} }`

**Edit detail of logged in user**
----
  Edit a user's details

* **URL**

  /user

* **Permission**

  `UserEdit` This permission is checked for logged in user. User without this permission may be able to see their details but not edit them.

* **Method:**

  `POST`
  
*  **URL Params**
  
   **Required:**
   
   `AuthToken[string]`
   
   **Optional**
 
   `OldPasswd[string]` This is required in case you're changing Passwd
   `Passwd=[string]` Greater than 5 chars
   `Name=[string]`
   `Email=[string]`

* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [],  Request: {..}} }`
 
* **Error Response:**

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {...} }`

  OR

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Wrong old password."], Request : {URL: "/user/BA89348", Passwd: "mynewpassword", OldPasswd: "myoldpasswd"} }`

  OR

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Password must be longer than 5 characters."], Request : {URL: "/user/BA89348", Passwd: "mynewpassword", OldPasswd: "myoldpasswd"} }`

  OR

  * **Code:** 401 UNAUTHORIZED <br />
    **Content:** `{ Errors : ["You're not authorized to perform this action."], Request : {...} }`

**Authenticate a user**
----
  Submit password and get authentication token for requests that require user authentication.

* **URL**

  /user/auth

* **Permission**

  None

* **Method:**

  `POST`
  
*  **URL Params**
  
   **Required:**
   
   `Passwd[string]`
   `Username[string]`

* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [], AuthToken: "72367BADBFFB8782378",  Request: {..}} }`
 
* **Error Response:**

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {...} }`

  OR

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Username doesn't exist or password is wrong."], Request : {...} }`


**Log out current user's token**
----
  Invalidates current authentication token of a user

* **URL**

  /user/logout

* **Permission**

  None

* **Method:**

  `POST`
  
*  **URL Params**
  
   **Required:**
   
   `AuthToken[string]`

* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [],  Request: {..}} }`
 
* **Error Response:**

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {...} }`

  OR

  * **Code:** 401 UNAUTHORIZED <br />
    **Content:** `{ Errors : ["User doesn't exist or token is wrong."], Request : {...} }`


**Log out all tokens of current user**
----
  Invalidates all authentication tokens of a user

* **URL**

  /user/logout/all

* **Permission**

  None

* **Method:**

  `POST`
  
*  **URL Params**
  
   **Required:**
   
   `AuthToken[string]`

* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [],  Request: {..}} }`
 
* **Error Response:**

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {...} }`

  OR

  * **Code:** 401 UNAUTHORIZED <br />
    **Content:** `{ Errors : ["User doesn't exist or token is wrong."], Request : {...} }`