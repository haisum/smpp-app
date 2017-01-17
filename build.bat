set cmd="git show -s --pretty=^"format:%%H^""

FOR /F "tokens=*" %%i IN (' %cmd% ') DO SET X=%%i
go build -o httpserver.exe -ldflags="-X main.version=%X%" .\utils\httpserver\  
go build -o smppworker -ldflags="-X main.version=%X%" .\utils\smppworker\
go build -o scheduler -ldflags="-X main.version=%X%" .\utils\scheduler\
go build -o soapservice -ldflags="-X main.version=%X%" .\utils\soapservice\