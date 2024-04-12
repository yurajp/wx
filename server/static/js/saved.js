
window.addEventListener("load", function(evt) {
  document.getElementById("chat").style.display="flex";
  var user = document.getElementById("uname").textContent;
  var addr = window.location.host;
  var cls = document.getElementById("cls");
  var clu = document.getElementById("clu");
  var out = document.getElementById("output");
  var urc = "https://" + addr + "/clear?user=" + user;
  var uru = "https://" + addr + "/unread?user=" + user;
  
  clu.onclick = (e) => {
    fetch(uru);
    clu.style.visibility="hidden";
  };
  cls.onclick = (e) => {
    fetch(urc);
    out.innerHTML = "";
    cls.style.visibility="hidden";
  };
  
  var remid = (mid) => {
    var lk = "https://" + window.location.host + "/unsave?mid=" + mid + "&user=" + user;
    fetch(lk);
  };
  
  var sels = document.getElementsByClassName('sel');
  for (let sp of sels) {
    sp.addEventListener('click', (e)=>{
      var par = e.target.parentNode;
      par.innerHTML="";
      remid(par.id);
      e.preventDefault();
    });
  }
  
  document.getElementById("input_img").addEventListener('change', function(evt) {
    var avaform = document.getElementById('avaform');
    const url = "https://" + window.location.host + "/newavatar";
    var data = new FormData(avaform);
    data.append("from", user);
    const fetchOpts = {
        method: 'POST',
        body: data,
    };
    fetch(url, fetchOpts);
    
    return false;
  });
  
});