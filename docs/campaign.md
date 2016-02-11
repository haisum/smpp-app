**Campaigns - send messages to multiple numbers in file**
----
  Sends message to all  numbers in a file

* **URL**

  /campaign

* **Method:**

  `POST`
  
*  **URL Params**

   **Required:**
 
   `File=[string]` ID of file to use as returned by /file endpoint
   `Src=[string]` 
   `Enc=[string]` Can be either latin or ucs
   `Priority=[int]` Can be a number from 0-9. Higher numbers are given priority 9 
   `Msg=[string]`
* **Data Params**

  None

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [], ID: "3267ACB67", Request : {File: "00BYUJKJ7785HGGH", Src: "+023032", Msg: "Hello", Priority: 1, Enc: "latin"} }`
 
* **Error Response:**

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Message is empty.", "Destination file doesn't exist.", "Source is empty.", "Encoding can either be \"latin\" or \"ucs\"."], ID: "",  Request : {File: "00BYUJKJ7785HGGH", Src: "", Msg: "", Priority: 0, Enc: "latin1"} }`

  OR

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `{ Errors : ["Internal server error. See logs for details."], ID: "",  Request : {File: "00BYUJKJ7785HGGH", Src: "", Msg: "hello", Priority: 0, Enc: "latin"} }`

* **Sample Call:** <br/>
  **Javascript**
  ```javascript
    $.ajax({
      url: "/campaign",
      dataType: "json",
      type : "POST",
      data : {Priority: 1, File: "00BYUJKJ7785HGGH", Src: "+93299", Msg: "hello world", Enc:"latin"}
      success : function(r) {
        console.log(r);
      }
    });
  ```
  **Curl**
  ```shell
  curl -k  --data-urlencode "Msg=Hello world" --data-urlencode "Src=+97153434"  --data-urlencode "Priority=1" --data-urlencode "File=00BYUJKJ7785HGGH" --data-urlencode "Enc=latin" https://localhost:8443/api/campaign
  ```
OR

  ```shell
  curl -k  --data-urlencode "Msg=لتنشيطالتنشيطالتنشيط التنشيط التنشيط اللتنشيطالتنشيطالتنشيط التنشيط االتنشيطالتنشيطالتنشيط التنشيط التنشيط اللتنشيطالتنشيطالتنشيط التنشيط اا" --data-urlencode "Src=+97153434"  --data-urlencode "Priority=1" --data-urlencode "File=00BYUJKJ7785HGGH" --data-urlencode "Enc=ucs" https://localhost:8443/api/campaign
  ```
