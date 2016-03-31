
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
		$(".button-collapse").sideNav();
		$('.datepicker').pickadate({
		    selectMonths: true, // Creates a dropdown to control month
		    selectYears: 15 // Creates a dropdown of 15 years to control year
		});
		$('.timepicker').pickatime({
	      twelvehour: true
	    });
	    $('select').material_select();
	}
}


$(app.init);