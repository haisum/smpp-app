**Send message to a number**
----
  Send message to given destination number

* **URL**

  /message

* **Method:**

  `POST`
  
*  **URL Params**

   **Required:**
 
   `Dst=[string]`
   `Src=[string]` 
   `Enc=[string]` Can be either latin or ucs
   `Priority=[int]` Can be a number from 0-9. Higher numbers are given priority 9 
   `Msg=[string]`
* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [], ID: "34534ADFBBCC", Request : {URL: "/message", Dst: "+001932", Src: "+023032", Msg: "Hello", Priority: 1, Enc: "latin"} }`
 
* **Error Response:**

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Message is empty.", "Destination is empty or file couldn't be loaded.", "Source is empty.", "Encoding can either be \"latin\" or \"ucs\"."], Request : {URL: "/message", Dst: "", ID: "", Src: "", Msg: "", Priority: 0, Enc: "latin1"} }`

  OR

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {URL: "/message", Dst: "", ID: "", Src: "", Msg: "", Priority: 0, Enc: "latin1"} }`

  OR

  * **Code:** 401 UNAUTHORIZED <br />
    **Content:** `{ Errors : ["You're not authorized to perform this action."], Request : {URL: "/message", Dst: "", ID: "", Src: "", Msg: "", Priority: 0, Enc: "latin1"} }`

* **Sample Call:** <br/>
  **Javascript**
  ```javascript
    $.ajax({
      url: "/message",
      dataType: "json",
      type : "POST",
      data : {Priority: 1, Dst: "+030032", Src: "+93299", Msg: "hello world", Enc:"latin"}
      success : function(r) {
        console.log(r);
      }
    });
  ```
  **Curl**
  ```shell
  curl -k  --data-urlencode "Msg=Hello world" --data-urlencode "Src=+97153434"  --data-urlencode "Priority=1" --data-urlencode "Dst=+971" --data-urlencode "Enc=latin" https://localhost:8443/api/message
  ```
OR

  ```shell
  curl -k  --data-urlencode "Msg=لتنشيطالتنشيطالتنشيط التنشيط التنشيط اللتنشيطالتنشيطالتنشيط التنشيط االتنشيطالتنشيطالتنشيط التنشيط التنشيط اللتنشيطالتنشيطالتنشيط التنشيط اا" --data-urlencode "Src=+97153434"  --data-urlencode "Priority=1" --data-urlencode "Dst=+971" --data-urlencode "Enc=ucs" https://localhost:8443/api/message
  ```
  
**Check message status**
----
  Get status of a sent message

* **URL**

  /message/:id

* **Method:**

  `GET`
  
*  **URL Params**
  
  None
  
* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [], ID: "34534ADFBBCC", Status: "", SentAt: "", SentBy: "", DeliveredAt: "", ErrorAt: "", CancelledAt: "", CancelledBy: "", Src: "", Dst: "", Msg: "", Enc: "", Conn: "du-1",  Priority: 0,  Request : {URL: "/message/11"} }`
 
* **Error Response:**

  * **Code:** 404 NOT FOUND <br />
    **Content:** `{ Errors : ["Requested message not found"], Request : {URL: "/message/10"} }`

  OR

  * **Code:** 401 UNAUTHORIZED <br />
    **Content:** `{ Errors : ["You're not authorized to perform this action."], Request : {URL: "/message/10"} }`

  OR

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {URL: "/message/2387"} }`

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

* **Method:**

  `POST`
  
*  **URL Params**
  
  None
  
* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [], Request : {URL: "/message/11/stop"} }`
 
* **Error Response:**

  * **Code:** 404 NOT FOUND <br />
    **Content:** `{ Errors : ["Requested message not found"], Request : {URL: "/message/10/stop"} }`

  OR

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Message already sent"], Request : {URL: "/message/10/stop"} }`

  OR

  * **Code:** 401 UNAUTHORIZED <br />
    **Content:** `{ Errors : ["You're not authorized to perform this action."], Request : {URL: "/message/10/stop"} }`

  OR

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], Request : {URL: "/message/2387/stop"} }`

* **Sample Call:** <br/>
  **Javascript**
  ```javascript
    $.ajax({
      url: "/message/1/stop",
      dataType: "json",
      type : "POST",
      success : function(r) {
        console.log(r);
      }
    });
  ```
  **Curl**
  ```shell
  curl -k  https://localhost:8443/api/message/11/stop
  ```