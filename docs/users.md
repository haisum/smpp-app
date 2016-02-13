**See possible permissions**
----
  See all possible permissions for users

* **URL**

  /users/permissions

* **Permission**

  `UsersPermissions`

* **Method:**

  `GET`
  
* **URL Params**
  
  **Required:**
  
  `AuthToken[string]`

* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [], Permissions: ["MessageSend", "MessageStop", "MessageRetry", "MessageStatus", "CampaignSend", "CampaignStatus", "CampaignRetry", "CampaignStop", ...], Request: {..}} }`
 
* **Error Response:**

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {...} }`

  OR

  * **Code:** 401 UNAUTHORIZED <br />
    **Content:** `{ Errors : ["You're not authorized to perform this action."], Request : {...} }`


**Add a user**
----
  Add a user with username, email, password and permissions

* **URL**

  /users/add

* **Permission**

  `UsersAdd`

* **Method:**

  `POST`
  
*  **URL Params**

   **Required:**
  
   `AuthToken[string]`
   `Username=[string]` Must be alphanumeric only
   `Passwd=[string]` Greater than 5 chars
   `Permissions=[[]string]` Array of permissions granted to this user. Possible permissions can be obtained from /user/permissions endpoint
   
   **Optional**
   
   `Name=[string]`
   `Email=[string]`
   
* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [], Username: "34534ADFBBCC", Request : {URL: "/users/add", Username: "34534ADFBBCC", Permissions: ["MessageSend", "MessageStop", "MessageRetry", "MessageStatus", "CampaignSend", "CampaignStatus", "CampaignRetry", "CampaignStop"]} }`
 
* **Error Response:**

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Username already exists"], Request : {...} }`

  OR

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Username is empty.", "Password is empty."], Request : {...} }`

  OR

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Username contains non-alphanumeric characters."], Request : {...} }`

  OR

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Password must be more than 5 chars long."], Request : {...} }`

  OR

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {...} }`

  OR

  * **Code:** 401 UNAUTHORIZED <br />
    **Content:** `{ Errors : ["You're not authorized to perform this action."], Request : {...} }`

**Edit a user**
----
  Edit a user's details and permissions

* **URL**

  /users/edit

* **Permission**

  `UserAdd`

* **Method:**

  `POST`
  
*  **URL Params**

   **Required:**
  
   `AuthToken[string]`
   `Username=[string]` Must be alphanumeric only
   `Passwd=[string]` Greater than 5 chars
   `Permissions=[[]string]` Array of permissions granted to this user. Possible permissions can be obtained from /user/permissions endpoint
   
   **Optional**
   
   `Name=[string]`
   `Email=[string]`
   
* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [], Username: "34534ADFBBCC", Request : {URL: "/users/add", Username: "34534ADFBBCC", Permissions: ["MessageSend", "MessageStop", "MessageRetry", "MessageStatus", "CampaignSend", "CampaignStatus", "CampaignRetry", "CampaignStop"]} }`
 
* **Error Response:**

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Username already exists"], Request : {...} }`

  OR

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Username is empty.", "Password is empty."], Request : {...} }`

  OR

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Username contains non-alphanumeric characters."], Request : {...} }`

  OR

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Password must be more than 5 chars long."], Request : {...} }`

  OR

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {...} }`

  OR

  * **Code:** 401 UNAUTHORIZED <br />
    **Content:** `{ Errors : ["You're not authorized to perform this action."], Request : {...} }`