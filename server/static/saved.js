
window.addEventListener("load", function(evt) {
  
  document.getElementById("chat").style.display="flex";
  
  var user = document.getElementById("username").textContent;
  var addr = document.getElementById("addr").textContent;
  var clear = document.getElementById("open");
  var out = document.getElementById("output");
  var url = "https://" + addr + "/clear?user=" + user;
  
  clear.onclick = (e) => {
    fetch(url);
    out.innerHTML = "";
  };
  
  
});