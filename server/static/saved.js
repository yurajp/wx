
window.addEventListener("load", function(evt) {
  
  document.getElementById("chat").style.display="flex";
  
  var user = document.getElementById("uname").textContent;
  var addr = window.location.host;
  var cls = document.getElementById("cls");
  var out = document.getElementById("output");
  var url = "https://" + addr + "/clear?user=" + user;
  var uru = "https://" + addr + "/unread?user=" + user;
  
  clu.onclick = (e) => {
    fetch(uru);
  };
  cls.onclick = (e) => {
    fetch(url);
    out.innerHTML = "";
  };

});