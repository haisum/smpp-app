**Send message to a number**
----
  Send message to given destination number. If text of message is larger than maximum characters allowed in a given encoding, split message and send as separate messages and return IDs.

* **URL**

  /message

* **Permission**

  `MessageSend`

* **Method:**

  `POST`
  
*  **URL Params**

   **Required:**
 
   `Dst=[string]`
   `Src=[string]` 
   `Enc=[string]` Can be either latin or ucs
   `Priority=[int]` Can be a number from 0-9. Higher numbers are given priority 9 
   `Msg=[string]`
   `AuthToken[string]`
   
   **Optional**
   
   `SendAt=[string]` Time to schedule this message for in YYYY-MM-DD HH:MM:SS format
   
* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [], IDs: ["34534ADFBBCC", "BACC87867A"], Request : {URL: "/message", SendAt: "", Dst: "+001932", Src: "+023032", Msg: "Hello", Priority: 1, Enc: "latin", IDs: []} }`
 
* **Error Response:**

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Message is empty.", "Destination is empty or file couldn't be loaded.", "Source is empty.", "Encoding can either be \"latin\" or \"ucs\"."], Request : {URL: "/message", SendAt: "", Dst: "", IDs: [], Src: "", Msg: "", Priority: 0, Enc: "latin1"} }`

  OR

* **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["SendAt is time in past or is not in format YYYY-MM-DD HH:MM:SS"], Request : {URL: "/message", SendAt: "2010-09-09 3:45:44", Dst: "", IDs: [], Src: "", Msg: "", Priority: 0, Enc: "latin1"} }`

  OR

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {URL: "/message", SendAt: "", Dst: "", IDs: [], Src: "", Msg: "", Priority: 0, Enc: "latin1"} }`

  OR

  * **Code:** 401 UNAUTHORIZED <br />
    **Content:** `{ Errors : ["You're not authorized to perform this action."], Request : {URL: "/message", SendAt: "", Dst: "", IDs: [], Src: "", Msg: "", Priority: 0, Enc: "latin1"} }`

* **Sample Call:** <br/>
  **Javascript**
  ```javascript
    $.ajax({
      url: "/message",
      dataType: "json",
      type : "POST",
      data : {Priority: 1, SendAt: "2016-11-30 12:00:00", Dst: "+030032", Src: "+93299", Msg: "hello world", Enc:"latin"}
      success : function(r) {
        console.log(r);
      }
    });
  ```
  **Curl**
  ```shell
  curl -k -X POST  --data-urlencode "Msg=Hello world" --data-urlencode "Src=+97153434"  --data-urlencode "Priority=1" --data-urlencode "Dst=+971" --data-urlencode "Enc=latin" https://localhost:8443/api/message
  ```

  OR

  ```shell
  curl -k -X POST --data-urlencode "Msg=لتنشيطالتنشيطالتنشيط التنشيط التنشيط اللتنشيطالتنشيطالتنشيط التنشيط االتنشيطالتنشيطالتنشيط التنشيط التنشيط اللتنشيطالتنشيطالتنشيط التنشيط اا" --data-urlencode "Src=+97153434"  --data-urlencode "Priority=1" --data-urlencode "Dst=+971" --data-urlencode "Enc=ucs" https://localhost:8443/api/message
  ```
  
  OR

  ```shell
  curl -k -X POST --data-urlencode "Msg=Scheduled Hello world" --data-urlencode "Src=+97153434" --data-urlencode "Src=+97153434"  --data-urlencode "Priority=1" --data-urlencode "Dst=+971" --data-urlencode "Enc=ucs" https://localhost:8443/api/message
  ```
  
**Check message status**
----
  Get status of a sent message

* **URL**

  /message/:id

* **Permission**

  `MessageStatus`

* **Method:**

  `GET`
  
*  **URL Params**
   
   **Required**

   `AuthToken[string]`
  
* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [], ID: "34534ADFBBCC", Status: "", SentAt: "", SentBy: "", DeliveredAt: "", ErrorAt: "", CancelledAt: "", CancelledBy: "", Src: "", Dst: "", Msg: "", Enc: "", Conn: "du-1",  Priority: 0,  Request : {URL: "/message/BACC87867A"} }`
 
* **Error Response:**

  * **Code:** 404 NOT FOUND <br />
    **Content:** `{ Errors : ["Requested message not found"], Request : {URL: "/message/AACC767A"} }`

  OR

  * **Code:** 401 UNAUTHORIZED <br />
    **Content:** `{ Errors : ["You're not authorized to perform this action."], Request : {URL: "/message/AACC767A"} }`

  OR

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {URL: "/message/AACC767A"} }`

* **Sample Call:** <br/>
  **Javascript**
  ```javascript
    $.ajax({
      url: "/message/1",
      dataType: "json",
      type : "GET",
      success : function(r) {
        console.log(r);
      }
    });
  ```
  **Curl**
  ```shell
  curl -k  https://localhost:8443/api/message/11
  ```
 
 **Stop message**
----
  Stop a message that has not been sent yet 

* **URL**

  /message/:id/stop

* **Permission**

  `MessageStop`

* **Method:**

  `POST`
  
*  **URL Params**
   
   **Required**

   `AuthToken[string]`
  
* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [], Request : {URL: "/message/AACC767A/stop"} }`
 
* **Error Response:**

  * **Code:** 404 NOT FOUND <br />
    **Content:** `{ Errors : ["Requested message not found"], Request : {URL: "/message/AACC767A/stop"} }`

  OR

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Message already sent"], Request : {URL: "/message/AACC767A/stop"} }`

  OR

  * **Code:** 401 UNAUTHORIZED <br />
    **Content:** `{ Errors : ["You're not authorized to perform this action."], Request : {URL: "/message/AACC767A/stop"} }`

  OR

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {URL: "/message/AACC767A/stop"} }`

* **Sample Call:** <br/>
  **Javascript**
  ```javascript
    $.ajax({
      url: "/message/AACC767A/stop",
      dataType: "json",
      type : "POST",
      success : function(r) {
        console.log(r);
      }
    });
  ```
  **Curl**
  ```shell
  curl -k  https://localhost:8443/api/message/AACC767A/stop
  ```

**Retry message**
----
  Retry a message that had failed earlier or hasn't been sent yet 

* **URL**

  /message/:id/retry

* **Method:**

  `POST`

* **Permission**

  `MessageRetry`
  
*  **URL Params**
   
   **Required**

   `AuthToken[string]`
  
* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [], Request : {URL: "/message/AACC767A/retry"} }`
 
* **Error Response:**

  * **Code:** 404 NOT FOUND <br />
    **Content:** `{ Errors : ["Requested message not found"], Request : {URL: "/message/AACC767A/retry"} }`

  OR

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Message already sent"], Request : {URL: "/message/AACC767A/retry"} }`

  OR

  * **Code:** 401 UNAUTHORIZED <br />
    **Content:** `{ Errors : ["You're not authorized to perform this action."], Request : {URL: "/message/AACC767A/retry"} }`

  OR

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {URL: "/message/AACC767A/retry"} }`

* **Sample Call:** <br/>
  **Javascript**
  ```javascript
    $.ajax({
      url: "/message/AACC767A/retry",
      dataType: "json",
      type : "POST",
      success : function(r) {
        console.log(r);
      }
    });
  ```
  **Curl**
  ```shell
  curl -k  https://localhost:8443/api/message/AACC767A/retry
  ```