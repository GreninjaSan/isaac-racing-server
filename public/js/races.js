$(document).ready(function () {
	ConvertTimeRaceStamps();
});

function ConvertTimeRaceStamps() {
	var m_names = new Array("Jan", "Feb", "Mar", "Apr", "May", "June", "July", "Aug", "Sept", "Oct", "Nov", "Dec");
	var d_names = new Array("Sun", "Mon", "Tue", "Wed", "Thur", "Fri", "Sat");

	$('#race-listing-table tr').each(function(){
		if ($(this).find('td').is('#racedate')) {
			dt = new Date($(this).find("#racedate").eq(0).html());
			console.log(dt);
			var curr_hours = dt.getHours();
			var curr_min = dt.getMinutes();
			var curr_time = curr_hours + ":" + ((curr_min < 10) ? "0" + curr_min : curr_min);
			var curr_date = dt.getDate();
			var sup = "";
			if (curr_date == 1 || curr_date == 21 || curr_date == 31)
			   {
			   sup = "st";
			   }
			else if (curr_date == 2 || curr_date == 22)
			   {
			   sup = "nd";
			   }
			else if (curr_date == 3 || curr_date == 23)
			   {
			   sup = "rd";
			   }
			else
			   {
			   sup = "th";
			   }
			 
			 console.log(d_names[dt.getDay()] + ", " + m_names[dt.getMonth()] + " " + curr_date + sup + ", " + curr_time);
						
			$(this).find("td#racedate").eq(0).html(d_names[dt.getDay()] + ", " + m_names[dt.getMonth()] + " " + curr_date + sup + ", " + curr_time);
		}
	});
};
