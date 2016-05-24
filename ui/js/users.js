$.extend(app, {
	renderUsers : function(){
	    if (!app.headerRendered) {
	        app.renderHeader(app.renderUsers);
	        app.headerRendered = true;
	        return;
	    }
	    $(".menuitem").removeClass("active");
	    $(".menuitem.users").addClass("active");
	    $("#page-title").html("Users");
	    $.ajax("/templates/users.html").done(function(data){
	        $("#inner-content").html(data);
	        $('ul.tabs').tabs();
	        $('.datepicker').pickadate({
	            selectMonths: true, // Creates a dropdown to control month
	            selectYears: 15 // Creates a dropdown of 15 years to control year
	        });
	        $('.timepicker').pickatime({
	          twelvehour: false
	        });
	        app.renderPermissionsSelect();
	        $("#userfilter-form").on("submit", function(e){
	            e.preventDefault();
	            var userData = {
	                ConnectionGroup : $("#ConnectionGroup").val(),
	                Username : $("#Username").val(),
	                RegisteredBefore    : utils.dateFieldToEpoch("RegisteredBefore"),
	                RegisteredAfter    : utils.dateFieldToEpoch("RegisteredAfter"),
	                Email: $("#Email").val(),
	                Name: $("#Name").val(),
	                Suspended: ($("#Suspended").val() == "true"),
	                Permissions: $("#userfilter-form select.Permissions").val(),
	                PerPage: parseInt($("#PerPage").val()),
	                From: $("#From").val(),
	                OrderByKey: $("#OrderByKey").val(),
	                OrderByDir: $("#OrderByDir").val(),
	                Token : localStorage.getItem("auth_token")
	            };
	            $.ajax({
	                url : "/api/users",
	                data : JSON.stringify(userData),
	                type : "POST",
	                dataType : "json",
	                contentType: "application/json",
	            }).done(function(data){
	                Materialize.toast("Users filtered.", 5000);
	                var source   = $("#users-template").html();
	                var template = Handlebars.compile(source);
	                var html    = template(data.Response);
	                $("#users").html(html);
	            }).fail(function(xhr, status, errThrone){
	                if(xhr.status == 401) {
	                    localStorage.removeItem("auth_token");
	                    window.location.reload();
	                }
	                utils.showErrors(xhr.responseJSON.Errors);
	            });
	            return false;
	        });
	        $("#useradd-form").on("submit", function(e){
	            e.preventDefault();
	            var userData = {
	                ConnectionGroup : $("#addConnectionGroup").val(),
	                Username : $("#addUsername").val(),
	                Password : $("#addPassword").val(),
	                Email: $("#addEmail").val(),
	                Name: $("#addName").val(),
	                Suspended: $("#addSuspended").prop("checked"),
	                Permissions: $("#useradd-form select.Permissions").val(),
	                Token : localStorage.getItem("auth_token"),
	            };
	            $.ajax({
	                url : "/api/users/add",
	                data : JSON.stringify(userData),
	                dataType : "json",
	                type : "post",
	                contentType : "application/json",
	            }).done(function(data){
	                Materialize.toast("User added.", 5000);
	                $("#useradd-form input").val("");
	            }).fail(function(xhr, status, errThrone){
	                if(xhr.status == 401) {
	                    localStorage.removeItem("auth_token");
	                    window.location.reload();
	                }
	                utils.showErrors(xhr.responseJSON.Errors);
	            });
	            return false;
	        });
	        $("#finduser-form").on("submit", function(e){
	        	e.preventDefault();
	        	$.ajax({
	        		url : "/api/users",
	        		data : {
	        			Username : $("#findUsername").val(),
	        			Token: localStorage.getItem("auth_token"),
	        		},
	        		dataType : "json"
	        	}).done(function(data){
	        		if (data.Response.Users && data.Response.Users.length == 1) {
	        			Materialize.toast("User found", 5000);
	        			$("#editName").val(data.Response.Users[0].Name).change();
	        			$("#useredit-form select.Permissions").val(data.Response.Users[0].Permissions);
	        			$('select').material_select();
	        			$("#editEmail").val(data.Response.Users[0].Email).change();
	        			$("#editConnectionGroup").val(data.Response.Users[0].ConnectionGroup).change();
	        			$("#editSuspended").prop("checked", data.Response.Users[0].Suspended);
	        		} else {
	        			var toastContent = '<span class="red-text">Couldn\'t find user.</span>';
		                Materialize.toast(toastContent, 5000);	
	        		}
	        	}).fail(function(xhr, status, errThrone){
	                if(xhr.status == 401) {
	                    localStorage.removeItem("auth_token");
	                    window.location.reload();
	                }
	                utils.showErrors(xhr.responseJSON.Errors);
	            });
	        	return false;

	        });
	        $("#useredit-form").on("submit", function(e){
	            e.preventDefault();
	            var userData = {
	                ConnectionGroup : $("#editConnectionGroup").val(),
	                Password : $("#editPassword").val(),
	                Email: $("#editEmail").val(),
	                Username : $("#findUsername").val(),
	                Name: $("#editName").val(),
	                Suspended: $("#editSuspended").prop("checked"),
	                Permissions: $("#useredit-form select.Permissions").val(),
	                Token : localStorage.getItem("auth_token"),
	            };
	            $.ajax({
	                url : "/api/users/edit",
	                data : JSON.stringify(userData),
	                dataType : "json",
	                type : "post",
	                contentType : "application/json",
	            }).done(function(data){
	                Materialize.toast("User updated.", 5000);
	                $("#useredit-form input, #finduser-form input").val("");
	            }).fail(function(xhr, status, errThrone){
	                if(xhr.status == 401) {
	                    localStorage.removeItem("auth_token");
	                    window.location.reload();
	                }
	                utils.showErrors(xhr.responseJSON.Errors);
	            });
	            return false;
	        });
	    });
	},
	renderPermissionsSelect : function(){
	    var data = {
	        Token: localStorage.getItem("auth_token"),
	        Username: app.userInfo.Username
	    }
	    $.ajax({
	        url : "/api/users/permissions",
	        data : data,
	        dataType: "json",
	        type: "get"
	    }).done(function(data){        
	        var source   = $("#Permissions-template").html();
	        var template = Handlebars.compile(source);
	        var html    = template(data.Response);
	        $(".PermissionsSelect").html(html);
	        $('select').material_select();
	    }).error(function(data){
	        if(xhr.status == 401) {
	            localStorage.removeItem("auth_token");
	            window.location.reload();
	        }
	        utils.showErrors(xhr.responseJSON.Errors);
	    });
	}
});