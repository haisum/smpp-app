Handlebars.registerHelper('prettyDate', function(unixDate) {
    if (unixDate == 0 || isNaN(unixDate) || typeof unixDate === undefined){
        return "";
    }
    var months = "Jan,Feb,Mar,Apr,May,Jun,Jul,Aug,Sep,Oct,Nov,Dec";
    function nth(d) {
      if(d>3 && d<21) return 'th'; // thanks kennebec
      switch (d % 10) {
            case 1:  return "st";
            case 2:  return "nd";
            case 3:  return "rd";
            default: return "th";
        }
    }
    d = new Date(1000 * unixDate);
    return d.getDate() + nth(d.getDate()) + " " + months.split(",")[d.getMonth()] + ", " + d.getFullYear() + " " + d.getHours() + ":" + d.getMinutes() + ":" + d.getSeconds();
});
// for stupid IE
$.ajaxSetup({ cache: false });
var app = {
    userInfo : {
        Username : ""
    },
    headerRendered : false,
    init : function(){
        if (localStorage.getItem("auth_token") == null ){
            app.renderLogin();
        } else {
            $.ajax({
                url : "/api/user/info",
                type: "get",
                dataType : "json",
                data: {Token : localStorage.getItem("auth_token")}
            }).done(function(data){
                app.userInfo = data.Response;
                var routes = {
                    "#!campaign" : app.renderCampaign,
                    "#!files" : app.renderFiles,
                    "#!reports": app.renderReports,
                    "#!users": app.renderUsers,
                    "#!services": app.renderServices
                };
                if (routes[window.location.hash]){
                    routes[window.location.hash]();
                } else {
                    app.renderSMS();
                }
                app.checkServices();
            }).fail(function(xhr, status, errThrone){
                if (xhr.status == 401){
                    localStorage.removeItem("auth_token");
                    app.renderLogin();
                } else {
                    utils.showErrors(xhr.responseJSON.Errors);
                }
            });
        }
    },
    renderLogin: function(){
        $("#page-center-css").remove();
        $('<link id="page-center-css">')
          .appendTo('head')
          .attr({type : 'text/css', rel : 'stylesheet'})
          .attr('href', '/css/page-center.css');
        $.ajax("/templates/login.html").done(function(data){
            $("#content").html(data);
            $("#login-form").on("submit", function(e){
                e.preventDefault();
                $.ajax({
                    "url": "/api/user/auth",
                    "dataType": "json",
                    "type": "POST",
                    "data": $(this).serialize(),
                }).done(function(data){
                    localStorage.setItem("auth_token", data.Response.Token);
                    app.init();
                }).fail(function(xhr, status, errThrone){
                    utils.showErrors(xhr.responseJSON.Errors);
                });
                return false;
            });
        }).fail(function(xhr, status, errThrone){
            utils.showErrors(xhr.responseJSON.Errors);
        });
    },
    renderSMS: function(){
        if (!app.headerRendered) {
            app.renderHeader(app.renderSMS);
            app.headerRendered = true;
            return;
        }
        $(".menuitem").removeClass("active");
        $(".menuitem.message").addClass("active");
        $("#page-title").html("Message");
        $.ajax("/templates/message.html").done(function(data){
            $("#inner-content").html(data);
            $(".button-collapse").sideNav();
            $('.datepicker').pickadate({
                selectMonths: true, // Creates a dropdown to control month
                selectYears: 15 // Creates a dropdown of 15 years to control year
            });
            $('.timepicker').pickatime({
              twelvehour: false
            });
            $('select').material_select();
            app.renderMessageList();
            $("#message-form").on("submit", function(e){
                e.preventDefault();
                $("#message-form").find("button[type=submit]").addClass("disabled").next(".preloader-wrapper").addClass("active");
                var msgReq = {
                    "Enc" : $("#Enc").prop("checked") ? "ucs" : "latin",
                    "Msg" : $("#Msg").val(),
                    "Dst" : $("#Dst").val(),
                    "SendAfter" : $("#SendAfter").val(),
                    "SendBefore" : $("#SendBefore").val(),
                    "ScheduledAt" : utils.dateFieldToEpoch("ScheduledAt"),
                    "Src" : $("#Src").val(),
                    "Priority" : parseInt($("#Priority").val()) > 0 ? parseInt($("#Priority").val()) : 0,
                    "Token" : localStorage.getItem("auth_token")
                }
                $.ajax({
                    "url": "/api/message",
                    "dataType": "json",
                    "type": "POST",
                    "data": msgReq,
                }).done(function(data){
                    $("#message-form").find("button[type=submit]").removeClass("disabled").next(".preloader-wrapper").removeClass("active");
                    Materialize.toast("Message sent succesfully.", 5000);
                    app.renderMessageList();
                }).fail(function(xhr, status, errThrone){
                    if(xhr.status == 401) {
                        localStorage.removeItem("auth_token");
                        window.location.reload();
                    }
                    utils.showErrors(xhr.responseJSON.Errors);
                    $("#message-form").find("button[type=submit]").removeClass("disabled").next(".preloader-wrapper").removeClass("active");
                });
                return false;
            });
        });
    },
    renderServices: function(){
        if (!app.headerRendered) {
            app.renderHeader(app.renderServices);
            app.headerRendered = true;
            return;
        }
        $(".menuitem").removeClass("active");
        $(".menuitem.services").addClass("active");
        $("#page-title").html("Services");
        $.ajax("/templates/services.html").done(function(data){
            $("#inner-content").html(data);
            $(".button-collapse").sideNav();
            $('select').material_select();
            $.get("/api/services/config", {"Token" : localStorage.getItem("auth_token")}, function(data){
                $("#Config").val("\n" + JSON.stringify(data["Response"], null, 4));
                $("#Config").trigger('keyup');
            });
            $("#services-form").on("submit", function(e){
                e.preventDefault();
                $("#services-form").find("button[type=submit]").addClass("disabled").next(".preloader-wrapper").addClass("active");
                var config
                try {
                    config = $.parseJSON($("#Config").val());
                } catch(e){
                    Materialize.toast("JSON not valid.", 5000);
                    return;
                }
                configReq = {
                    "Config" : config,
                    "Token" : localStorage.getItem("auth_token")
                };
                $.ajax({
                    "url": "/api/services/config",
                    "dataType": "json",
                    "type": "POST",
                    "contentType" : "application/json",
                    "data": JSON.stringify(configReq),
                }).done(function(data){
                    $("#services-form").find("button[type=submit]").removeClass("disabled").next(".preloader-wrapper").removeClass("active");
                    Materialize.toast("Config updated succesfully.", 5000);
                    window.setTimeout(function () {
                      app.checkServices();
                    }, 2000);
                }).fail(function(xhr, status, errThrone){
                    if(xhr.status == 401) {
                        localStorage.removeItem("auth_token");
                        window.location.reload();
                    }
                    utils.showErrors(xhr.responseJSON.Errors);
                    $("#services-form").find("button[type=submit]").removeClass("disabled").next(".preloader-wrapper").removeClass("active");
                });
                return false;
            });
        });
    },
    renderHeader: function (callBackFunc){
        $.ajax("/templates/header.html").done(function(data){
            $("#content").html(data);
            $("#user-fullname").html(app.userInfo.Name == "" ? app.userInfo.Username : app.userInfo.Name);
            callBackFunc();
        });
    },
    renderReports: function(){
        if (!app.headerRendered) {
            app.renderHeader(app.renderReports);
            app.headerRendered = true;
            return;
        }
        $(".menuitem").removeClass("active");
        $(".menuitem.reports").addClass("active");
        $("#page-title").html("Reports");
        $.ajax("/templates/reports.html").done(function(data){
            $("#inner-content").html(data);
            $('select').material_select();
            app.renderCampaignSelect();
            $('.datepicker').pickadate({
               selectMonths: true, // Creates a dropdown to control month
               selectYears: 15 // Creates a dropdown of 15 years to control year
            });
            $('.timepicker').pickatime({
               twelvehour: false
            });

            $("#reports-form").on("submit", function(e){
                e.preventDefault();
                $("#reports-form").find("button[type=submit]").addClass("disabled").siblings(".preloader-wrapper").addClass("active");
                $("#Stats").prop("disabled", true);
                var reportData = utils.getReportData();
                reportData["Token"] = localStorage.getItem("auth_token");
                reportData["Stats"] = $("#Stats").prop("checked")
                $.ajax({
                    url : "/api/message/filter",
                    data : reportData,
                    dataType : "json",
                    type : "get"
                }).done(function(data){
                    $("#reports-form").find("button[type=submit]").removeClass("disabled").siblings(".preloader-wrapper").removeClass("active");
                    Materialize.toast("Report generated.", 5000);
                    var source   = $("#report-template").html();
                    var template = Handlebars.compile(source);
                    var html    = template(data.Response);
                    $("#report").html(html);
                    $("#Stats").prop("disabled", false);
                }).fail(function(xhr, status, errThrone){
                    if(xhr.status == 401) {
                        localStorage.removeItem("auth_token");
                        window.location.reload();
                    }
                    $("#reports-form").find("button[type=submit]").removeClass("disabled").siblings(".preloader-wrapper").removeClass("active");
                    $("#Stats").prop("disabled", false);
                    utils.showErrors(xhr.responseJSON.Errors);
                });
                return false;
            });
            $("#reports-csv").on("click", function(e){
                e.preventDefault();
                var reportData = utils.getReportData();
                reportData["Token"] = localStorage.getItem("auth_token");
                reportData["CSV"] = true;
                reportData["TZ"] = "Asia/Dubai"
                window.open("/api/message/filter?" + $.param(reportData));
                return false;
            });

            $.get("/api/message/filter?Token="+ localStorage.getItem("auth_token") + "&" + $.param(utils.getReportData()));
        });
    },
    renderFiles: function(){
        if (!app.headerRendered) {
            app.renderHeader(app.renderFiles);
            app.headerRendered = true;
            return;
        }
        $(".menuitem").removeClass("active");
        $(".menuitem.files").addClass("active");
        $("#page-title").html("Files");
        $.ajax("/templates/files.html").done(function(data){
            $("#inner-content").html(data);
            app.renderFileList();
            $("#files-form").on("submit", function(e){
                e.preventDefault();
                $("#files-form").find("button[type=submit]").addClass("disabled").next(".preloader-wrapper").addClass("active");
                var formData = new FormData($(this)[0]);
                formData.append("Token", localStorage.getItem("auth_token"));
                $.ajax({
                    url: "/api/file/upload",
                    type: 'POST',
                    data: formData,
                    cache: false,
                    contentType: false,
                    processData: false
                }).done(function(data){
                    $("#files-form").find("button[type=submit]").removeClass("disabled").next(".preloader-wrapper").removeClass("active");
                    Materialize.toast("File uploaded succesfully.", 5000);
                    $("#files-form input").val("");
                    app.renderFileList();
                }).fail(function(xhr, status, errThrone){
                    if(xhr.status == 401) {
                        localStorage.removeItem("auth_token");
                        window.location.reload();
                    }
                    $("#files-form").find("button[type=submit]").removeClass("disabled").next(".preloader-wrapper").removeClass("active");
                    utils.showErrors(xhr.responseJSON.Errors);
                });
                return false;
            });
        });
    },
    renderFileList: function(){
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
            var source   = $("#list-files-template").html();
            var template = Handlebars.compile(source);
            var html    = template(data.Response);
            $("#list-files").html(html);
        }).fail(function(xhr, status, errThrone){
            if(xhr.status == 401) {
                localStorage.removeItem("auth_token");
                window.location.reload();
            }
            utils.showErrors(xhr.responseJSON.Errors);
        });
    },
    renderMessageList: function(){
        var data = {
            Token: localStorage.getItem("auth_token"),
            Username: app.userInfo.Username
        }
        $.ajax({
            url : "/api/message/filter",
            data : data,
            dataType: "json",
            type: "get"
        }).done(function(data){
            var source   = $("#list-message-template").html();
            var template = Handlebars.compile(source);
            var html    = template(data.Response);
            $("#list-message").html(html);
        }).fail(function(xhr, status, errThrone){
            if(xhr.status == 401) {
                localStorage.removeItem("auth_token");
                window.location.reload();
            }
            utils.showErrors(xhr.responseJSON.Errors);
        });
    },
    checkServices : function(){
      $.ajax({
        url : "/api/services/status",
        data : {
          Token : localStorage.getItem("auth_token"),
        },
        type : "get",
        dataType : "json"
      }).done(function(data){
        notOk = [];
        for(i=0; i<data.Response.length; i++){
          if (!data.Response[i].Ok) {
            notOk.push(data.Response[i].Program + " " + data.Response[i].Status);
          }
        }
        if (notOk.length != 0){
          var errHtml = "Following services are not working: <ul>";
          for (i=0; i<notOk.length;i++){
              errHtml +=  "<li>" + notOk[i]  + "</li>";
          }
          errHtml += "</ul>"
          var toastContent = '<div class="red-text">' + errHtml + '</div>';
          Materialize.toast(toastContent, 5000)
        } else {
          var toastContent = '<div class="green-text">All services are up and running</div>';
          Materialize.toast(toastContent, 5000)
        }
      }).fail(function(xhr, status, errThrone){
          if(xhr.status == 401) {
              localStorage.removeItem("auth_token");
              window.location.reload();
          }
      });
    }
}

var utils = {
    logout: function() {
        localStorage.removeItem("auth_token");
        window.location.reload();
    },
    getReportData: function (){
        var data = {
            ConnectionGroup : $("#ConnectionGroup").val(),
            Connection      : $("#Connection").val(),
            Username        : $("#Username").val(),
            Enc             : $("#Enc").val(),
            Dst             : $("#Dst").val(),
            Src             : $("#Src").val(),
            Msg             : $("#Msg").val(),
            Priority        : $("#Priority").val(),
            Total           : $("#Total").val(),
            QueuedBefore    : utils.dateFieldToEpoch("QueuedBefore"),
            QueuedAfter     : utils.dateFieldToEpoch("QueuedAfter"),
            SentBefore      : utils.dateFieldToEpoch("SentBefore"),
            SentAfter       : utils.dateFieldToEpoch("SentAfter"),
            DeliveredBefore : utils.dateFieldToEpoch("DeliveredBefore"),
            DeliveredAfter  : utils.dateFieldToEpoch("DeliveredAfter"),
            ScheduledAt     : utils.dateFieldToEpoch("ScheduledAt"),
            CampaignID      : $("#CampaignID").val(),
            Status          : $("#Status").val(),
            Error           : $("#Error").val(),
            OrderByKey      : $("#OrderByKey").val(),
            OrderByDir      : $("#OrderByDir").val(),
            From            : $("#From").val(),
            PerPage         : $("#PerPage").val(),
        };
        return data;

    },
    dateFieldToEpoch : function (fieldName){
        var date = $("#" + fieldName + "_date").val();
        var time = $("#" + fieldName + "_time").val();
        if (date == "") return 0;
        if (time == "") time = "00:00";
        var datetime = Date.parse(date + " " + time);
        var d = new Date(datetime);
        return d.getTime() / 1000;
    },
    showErrors : function(errors) {
        console.log("Following errors occured:");
        console.log(errors);
        var errHtml = "<ul>";
        for (i=0; i<errors.length;i++){
            errHtml +=  "<li>" + errors[i].Message  + "</li>";
        }
        errHtml += "</ul>"
        var toastContent = '<div class="red-text">' + errHtml + '</div>';
        Materialize.toast(toastContent, 5000)
    }
}
