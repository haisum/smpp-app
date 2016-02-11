**Send message to a number**
----
  Sends message to given destination number

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
    **Content:** `{ Errors : [], Request : {Dst: "+001932", Src: "+023032", Msg: "Hello", Priority: 1, Enc: "latin"} }`
 
* **Error Response:**

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Message is empty.", "Destination is empty or file couldn't be loaded.", "Source is empty.", "Encoding can either be \"latin\" or \"ucs\"."], Request : {Dst: "", Src: "", Msg: "", Priority: 0, Enc: "latin1"} }`

  OR

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `Internal server error. See logs for details.`

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