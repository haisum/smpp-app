
var app = {
	init : function(){
		if (localStorage.getItem("auth_token") == null ){
			app.renderLogin();
		} else {
			app.renderSMS();
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
					app.renderSMS();
				}).fail(function(xhr, status, errThrone){
					console.error(xhr);
					var toastContent = '<span class="red-text">' + xhr.responseJSON.Errors.auth + '</span>';
		  			Materialize.toast(toastContent, 5000)	
				});
			});
		}).fail(function(xhr, status, errThrone){
			console.error(xhr);
			var toastContent = '<span class="red-text">Getting templates/login.html. ' + xhr.responseText + '</span>';
  			Materialize.toast(toastContent, 5000)
		});
	},
	renderSMS: function(){
		$.ajax("/templates/message.html").done(function(data){
			$("#content").html(data);
			$(".button-collapse").sideNav();
			$('.datepicker').pickadate({
			    selectMonths: true, // Creates a dropdown to control month
			    selectYears: 15 // Creates a dropdown of 15 years to control year
			});
			$('.timepicker').pickatime({
		      twelvehour: true
		    });
		    $('select').material_select();
		    $("#message-form").on("submit", function(e){
				e.preventDefault();
				var msgReq = {
					"Enc" : $("#Enc").prop("checked") ? "ucs" : "latin",
					"Msg" : $("#Msg").val(),
					"Dst" : $("#Dst").val(),
					"Src" : $("#Src").val(),
					"Token" : localStorage.getItem("auth_token")
				}
				$.ajax({
					"url": "/api/message",
					"dataType": "json",
					"type": "POST",
					"data": msgReq,
				}).done(function(data){
					Materialize.toast("Message sent succesfully.", 5000);
				}).fail(function(xhr, status, errThrone){
					if(xhr.status == 401) {
						localStorage.removeItem("auth_token");
						window.location.reload();
					}
					console.error(xhr.responseJSON);
					var toastContent = '<span class="red-text">Error occured see console for details.</span>';
		  			Materialize.toast(toastContent, 5000)	
				});
			});
		});
	},
	renderServices: function(){
		$.ajax("/templates/services.html").done(function(data){
			$("#content").html(data);
			$(".button-collapse").sideNav();
		    $('select').material_select();
		    $.get("/api/services/config", {"Token" : localStorage.getItem("auth_token")}, function(data){
		    	$("#Config").val(JSON.stringify(data["Response"], null, 4));
		    	$("#Config").trigger('keyup');
		    });
		    $("#services-form").on("submit", function(e){
				e.preventDefault();
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
					Materialize.toast("Config updated succesfully.", 5000);
				}).fail(function(xhr, status, errThrone){
					if(xhr.status == 401) {
						localStorage.removeItem("auth_token");
						window.location.reload();
					}
					console.error(xhr.responseJSON);
					var toastContent = '<span class="red-text">Error occured see console for details.</span>';
		  			Materialize.toast(toastContent, 5000)	
				});
			});
		});
	}
}


$(app.init);