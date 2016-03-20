
var app = {
	init : function(){
		$(".button-collapse").sideNav();
		$('.datepicker').pickadate({
		    selectMonths: true, // Creates a dropdown to control month
		    selectYears: 15 // Creates a dropdown of 15 years to control year
		});
		$('.timepicker').pickatime({
	      twelvehour: true
	    });
	    $('select').material_select();
	},
}


$(app.init);