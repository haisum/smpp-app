$.extend(app, {
    renderCampaignList: function(){
        var data = {
            Token: localStorage.getItem("auth_token"),
            Username: app.userInfo.Username
        }
        $.ajax({
            url : "/api/campaign/filter",
            data : data,
            dataType: "json",
            type: "get"
        }).done(function(data){        
            var source   = $("#list-campaign-template").html();
            var template = Handlebars.compile(source);
            var html    = template(data.Response);
            $("#list-campaign").html(html);
        }).error(function(data){
            if(xhr.status == 401) {
                localStorage.removeItem("auth_token");
                window.location.reload();
            }
            console.error(xhr.responseJSON);
            var toastContent = '<span class="red-text">Error occured see console for details.</span>';
            Materialize.toast(toastContent, 5000)
        });
    },
    renderCampaignSelect: function(){
        var data = {
            Token: localStorage.getItem("auth_token"),
            Username: app.userInfo.Username
        }
        $.ajax({
            url : "/api/campaign/filter",
            data : data,
            dataType: "json",
            type: "get"
        }).done(function(data){        
            var source   = $("#CampaignId-template").html();
            var template = Handlebars.compile(source);
            var html    = template(data.Response);
            $("#CampaignIdSelect").html(html);
            $('select').material_select();
        }).error(function(data){
            if(xhr.status == 401) {
                localStorage.removeItem("auth_token");
                window.location.reload();
            }
            console.error(xhr.responseJSON);
            var toastContent = '<span class="red-text">Error occured see console for details.</span>';
            Materialize.toast(toastContent, 5000)
        });
    },
    renderCampaignFiles: function(){
        var data = {
            Token: localStorage.getItem("auth_token"),
            Username: app.userInfo.Username
        }
        $.ajax({
            url : "/api/file/filter",
            data : data,
            dataType: "json",
            type: "get"
        }).done(function(data){        
            var source   = $("#FileId-template").html();
            var template = Handlebars.compile(source);
            var html    = template(data.Response);
            $("#FileIdSelect").html(html);
            $('select').material_select();
        }).error(function(data){
            if(xhr.status == 401) {
                localStorage.removeItem("auth_token");
                window.location.reload();
            }
            console.error(xhr.responseJSON);
            var toastContent = '<span class="red-text">Error occured see console for details.</span>';
            Materialize.toast(toastContent, 5000)
        });
    },
    renderCampaign: function(){
        if (!app.headerRendered) {
            app.renderHeader(app.renderCampaign);
            app.headerRendered = true;
            return;
        }
        $(".menuitem").removeClass("active");
        $(".menuitem.campaign").addClass("active");
        $("#page-title").html("Campaign");
        $.ajax("/templates/campaign.html").done(function(data){
            $("#inner-content").html(data);
            $('.materialize-textarea').characterCounter();
            app.renderCampaignFiles();
            app.renderCampaignList();
            $("#campaign-form").on("submit", function(e){
                e.preventDefault();
                var campReq = {
                    "Enc" : $("#Enc").prop("checked") ? "ucs" : "latin",
                    "Msg" : $("#Msg").val(),
                    "FileId" : $("#FileId").val(),
                    "Src" : $("#Src").val(),
                    "Token" : localStorage.getItem("auth_token"),
                    "Description": $("#Description").val(),
                }
                $.ajax({
                    "url": "/api/campaign",
                    "dataType": "json",
                    "type": "POST",
                    "data": campReq,
                }).done(function(data){
                    Materialize.toast("Campaign started succesfully.", 5000);
                    app.renderCampaignList();
                }).fail(function(xhr, status, errThrone){
                    if(xhr.status == 401) {
                        localStorage.removeItem("auth_token");
                        window.location.reload();
                    }
                    console.error(xhr.responseJSON);
                    var toastContent = '<span class="red-text">Error occured see console for details.</span>';
                    Materialize.toast(toastContent, 5000)   
                });
                return false;
            });
        });
    },
});