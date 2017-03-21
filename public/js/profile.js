$(document).ready(function () {
  ConvertTimeStamps();
});

function ConvertTimeStamps() {
  dt = $('span#date').html()/1000;
  var d = new Date(0);
  d.setUTCSeconds(dt);
  $('span#date').html(d.getMonth() + '/' + d.getDate() + '/' + d.getFullYear());
};