**Save destination numbers file**
----
  Takes a file and saves it for use in /fmessage requests 

* **URL**

  /file

* **Method:**

  `POST` multipart/form-data
  
* **Data Params**
    
   `File=[File]` Text file attached against input element named File

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `{ Errors : [], Request : {File: "numbers.csv"} }`
 
* **Error Response:**

  * **Code:** 400 BAD REQUEST <br />
    **Content:** `{ Errors : ["Invalid file."], Request : {File: "numbers.csv"} }`

  OR

  * **Code:** 500 INTERNAL SERVER ERROR <br />
    **Content:** `Internal server error. See logs for details.`

* **Sample Call:** <br/>
  **Javascript**
  ```javascript
    var fd = new FormData();    
    fd.append( 'File', $("input[name=File]").files[0] );

    $.ajax({
    url: '/file',
    data: fd,
    processData: false,
    contentType: false,
    type: 'POST',
    success: function(data){
        alert(data);
    }
    });
  ```
  **Curl**
  ```shell
  curl -k  -F "File=@file" https://localhost:8443/api/file
  ```