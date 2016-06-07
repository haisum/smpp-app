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
            utils.showErrors(xhr.responseJSON.Errors);
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
            var source   = $("#CampaignID-template").html();
            var template = Handlebars.compile(source);
            var html    = template(data.Response);
            $("#CampaignIDSelect").html(html);
            $('select').material_select();
        }).error(function(data){
            if(xhr.status == 401) {
                localStorage.removeItem("auth_token");
                window.location.reload();
            }
            utils.showErrors(xhr.responseJSON.Errors);
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
            var source   = $("#FileID-template").html();
            var template = Handlebars.compile(source);
            var html    = template(data.Response);
            $("#FileIDSelect").html(html);
            $('select').material_select();
        }).error(function(data){
            if(xhr.status == 401) {
                localStorage.removeItem("auth_token");
                window.location.reload();
            }
            utils.showErrors(xhr.responseJSON.Errors);
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
            app.renderCampaignSelect();
            $('ul.tabs').tabs();
            $('.datepicker').pickadate({
                selectMonths: true, // Creates a dropdown to control month
                selectYears: 15 // Creates a dropdown of 15 years to control year
            });
            $('.timepicker').pickatime({
              twelvehour: false
            });
            $("#campaign-form").on("submit", function(e){
                e.preventDefault();
                $("#campaign-form").find("button[type=submit]").addClass("disabled").next(".preloader-wrapper").addClass("active");
                var campReq = {
                    "Enc" : $("#Enc").prop("checked") ? "ucs" : "latin",
                    "Msg" : $("#Msg").val(),
                    "FileID" : $("#FileID").val(),
                    "Priority" : parseInt($("#Priority").val()) > 0 ? parseInt($("#Priority").val()) : 0,
                    "Src" : $("#Src").val(),
                    "Token" : localStorage.getItem("auth_token"),
                    "Description": $("#Description").val(),
                    "SendAfter" : $("#SendAfter").val(),
                    "SendBefore" : $("#SendBefore").val(),
                    "ScheduledAt" : utils.dateFieldToEpoch("ScheduledAt"),
                }
                $.ajax({
                    "url": "/api/campaign",
                    "dataType": "json",
                    "type": "POST",
                    "data": campReq,
                }).done(function(data){
                    $("#campaign-form").find("button[type=submit]").removeClass("disabled").next(".preloader-wrapper").removeClass("active");
                    Materialize.toast("All messages for campaign have been queued.", 5000);
                    app.renderCampaignList();
                    app.renderCampaignSelect();
                }).fail(function(xhr, status, errThrone){
                    if(xhr.status == 401) {
                        localStorage.removeItem("auth_token");
                        window.location.reload();
                    }
                    $("#campaign-form").find("button[type=submit]").removeClass("disabled").next(".preloader-wrapper").removeClass("active");
                    utils.showErrors(xhr.responseJSON.Errors);
                });
                return false;
            });
            $("#stopcampaign").on("click", function(e){
                e.preventDefault();
                $("#stopcampaign").addClass("disabled").next(".preloader-wrapper").addClass("active");
                $.ajax({
                  url : "/api/campaign/stop",
                  type: 'post',
                  data : {
                    Token : localStorage.getItem("auth_token"),
                    CampaignID: $("#CampaignID").val()
                  },
                  dataType: 'json'
                }).done(function(data){
                    $("#stopcampaign").removeClass("disabled").next(".preloader-wrapper").removeClass("active");
                    Materialize.toast(data.Response.Count + " pending messages have been stopped.", 5000);
                    app.renderCampaignList();
                }).fail(function(xhr, status, errThrone){
                    if(xhr.status == 401) {
                        localStorage.removeItem("auth_token");
                        window.location.reload();
                    }
                    $("#stopcampaign").removeClass("disabled").next(".preloader-wrapper").removeClass("active");
                    utils.showErrors(xhr.responseJSON.Errors);
                });
                return false;
            });
            $("#campaignreport").on("click", function(e){
                e.preventDefault();
                $("#campaignreport").addClass("disabled").next(".preloader-wrapper").addClass("active");
                $.ajax({
                  url : "/api/campaign/report",
                  type: 'get',
                  data : {
                    Token : localStorage.getItem("auth_token"),
                    CampaignID: $("#CampaignID").val()
                  },
                  dataType: 'json'
                }).done(function(data){
                    $("#campaignreport").removeClass("disabled").next(".preloader-wrapper").removeClass("active");
                    var source   = $("#campaignreport-template").html();
                    var template = Handlebars.compile(source);
                    var html    = template(data.Response);
                    $("#report-container").html(html);
                }).fail(function(xhr, status, errThrone){
                    if(xhr.status == 401) {
                        localStorage.removeItem("auth_token");
                        window.location.reload();
                    }
                    $("#campaignreport").removeClass("disabled").next(".preloader-wrapper").removeClass("active");
                    utils.showErrors(xhr.responseJSON.Errors);
                });
                return false;
            });
            $("#retrycampaign").on("click", function(e){
                e.preventDefault();
                $("#retrycampaign").addClass("disabled").next(".preloader-wrapper").addClass("active");
                $.ajax({
                  url : "/api/campaign/retry",
                  type: 'post',
                  data : {
                    Token : localStorage.getItem("auth_token"),
                    CampaignID: $("#CampaignID").val()
                  },
                  dataType: 'json'
                }).done(function(data){
                    $("#retrycampaign").removeClass("disabled").next(".preloader-wrapper").removeClass("active");
                    Materialize.toast(data.Response.Count + " error messages have been re-queued.", 5000);
                    app.renderCampaignList();
                }).fail(function(xhr, status, errThrone){
                    if(xhr.status == 401) {
                        localStorage.removeItem("auth_token");
                        window.location.reload();
                    }
                    $("#retrycampaign").removeClass("disabled").next(".preloader-wrapper").removeClass("active");
                    utils.showErrors(xhr.responseJSON.Errors);
                });
                return false;
            });

        });
    },
});
