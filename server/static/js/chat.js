  function deleteMessage(sid) {
    let user = document.getElementById("uname").textContent;
    let msg = document.getElementById(sid);
    let urdl = 'https://'+window.location.host+"/delete?sid="+sid+"&user="+user;
    fetch(urdl)
    .then((res) => res.ok)
    .then((ok) => {
      if (ok) {
        msg.style.display='none';
      }
    });
  }

function setTo(p) {
  var person = document.getElementById("person");
  person.textContent = p;
  return false;
}

var quote = "";

function toQuote(e) {
  var q = document.getElementById(e);
  if (quote == e) {
    q.style.background='#CAC0A0';
    quote = "";
    return false;
  }
  quote = e;
  
  var inp = document.querySelector("#input");
  var qt = document.createElement("div");
  qt.className = "quoted";
  var str = document.createElement("div");
  str.className = "strip";
  qt.appendChild(str);
  var qbx = document.createElement("div");
  qbx.className = "qbox";
  
  var nq = document.createElement("div");
  nq.innerHTML = q.innerHTML;
  var oldq = nq.getElementsByClassName("quoted");
  if (oldq.length > 0) {
    nq.removeChild(oldq[0]);
  }
  var lks = nq.getElementsByClassName("links");
  if (lks.length > 0) {
    nq.removeChild(lks[0]);
  }
  
  qbx.innerHTML = nq.innerHTML;
  qt.appendChild(qbx);
  inp.insertBefore(qt, inp.firstChild);
  
  return false;
}


window.addEventListener("load", function(evt) {
    var user = document.getElementById("uname").textContent;
    var users = ["+All"];
    var output = document.getElementById("output");
    var usermenu = document.getElementById("usermenu");
    var person = document.getElementById("person");
    var input = document.getElementById("input");
    var home = window.location.host;
    var sound = new Audio('static/snd/message.mp3');
    var ws;
    let date = new Date("01 Jan");
    const opts = {
	    day: 'numeric', month: 'long'
    };
    var record = document.getElementById("record");
        
    var mediaRecorder;
    var voice = [];
    navigator.mediaDevices.getUserMedia({audio: true})
      .then(stream => {
        mediaRecorder = new MediaRecorder(stream);
      mediaRecorder.addEventListener("dataavailable",function(event) {
        voice.push(event.data);
      });
      
      mediaRecorder.addEventListener("stop", function() {
        let voiceBlob = new Blob(voice, {
          type: 'audio/wav'
        }); 
        const url = "https://"+window.location.host+"/record";
        let fd = new FormData();
        fd.append("from", user);
        fd.append("to", person.textContent);
        fd.append("quote", quote);
        fd.append('voice', voiceBlob);

        fetch(url, {
	        method: 'POST',
	        body: fd})
        .then((e) => {
	        voice = [];
	        quote = "";
	        input.innerHTML = "<pre contenteditable='true'></pre>";
        });
      });
    });  
    
    var muted = true;
    var bimg = document.getElementById("bimg");
    var bell = document.getElementById("bell");
    bell.addEventListener('click', function(e) {
      e.preventDefault();
      if (muted) {
        muted = false;
        bimg.style.filter = 'invert(90%)';
      } else {
        muted = true;
        bimg.style.filter = 'invert(20%)';
      }
    });
    
    var handleMessage = function(message) {
      if (!ws) {
        return false;
      }
      var ms = JSON.parse(message);
      if (ms.kind == "users") {
        var us = JSON.parse(ms.content);
        users = ["+All"];
        users.push( ...us);
        
        return false;
      }
      if (ms.kind != "html") {
        return false;
      }
      var dt = new Date(ms.date);
      if (dt > date) {
        date = dt;
        let dd = document.createElement("div");
        dd.className = "date";
        dd.textContent = dt.toLocaleDateString("ru-RU", opts);
        output.appendChild(dd);
      }
      var box = document.createElement("div");
      box.innerHTML = ms.content;
      
      var inner = box.getElementsByClassName("inbox")[0];
      var hd = inner.getElementsByClassName("header")[0];
      
      var tr = hd.getElementsByClassName("trash")[0];
      var fr = hd.getElementsByClassName("id-from")[0].textContent;
      var to = hd.getElementsByClassName("id-to")[0].textContent;
      if (to != user && fr != user) {
        tr.style.visibility = 'hidden';
      }
      
      output.appendChild(box);
      output.scroll(0, output.scrollHeight);
      
      return false;
    }; 
    
    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        ws = new WebSocket("wss://"+window.location.host+"/translator");
        ws.onopen = function(evt) {
          document.getElementById("chat").style.display='flex';
        };
        ws.onclose = function(evt) {
          ws.close(1001, "going away");
          ws = null;
          window.location.reload();
        };
        
        ws.onmessage = function(evt) {
          if (!muted) {
            sound.play();
          }
          handleMessage(evt.data);
        };
        ws.onerror = function(evt) {
            ws.close();
        };
        document.getElementById('open').style.display='none';
        document.getElementById('close').style.display='inline-block';
        return false;
    };

    document.getElementById("send").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        let to = person.textContent;
        let text = input.innerHTML;
        
        
        var js = JSON.stringify({from:user,to:to,type:"text",data:text,quote:quote});
        ws.send(js);
        person.textContent = "All";
        input.innerHTML = "<pre contenteditable='true'></pre>";
        if (!quote) {
          return false;
        }
        document.getElementById(quote).style.background='#CAC0A0';
        quote = "";
        return false;
    };

    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        window.location.reload();
        ws.close(1001, "going away");
        return false;
    };
    
    person.onclick = function(evt) {
      if (!users) {
        return false;
      }
      if (usermenu.style.display == "block") {
        usermenu.style.display = 'none';
        return false;
      }
      usermenu.innerHTML='';
      var ul = document.createElement("ul");
      users.forEach(function(item) {
        var li = document.createElement("li");
        var unm = item.substring(1);
        li.textContent = unm;
        if (item[0] == '+') {
          li.className = "online";
        } else {
          li.className = "offline";
        }
        li.addEventListener('click', (e)=>{
          let name = e.target.textContent;
          person.textContent = name;
          usermenu.style.display='none';
        });
        ul.appendChild(li);
      });
      usermenu.appendChild(ul);
      usermenu.style.display="block";
      
      return false;
    };
    
    document.getElementById("input_file").addEventListener('change', function(evt) {
      if (!ws) {
        return false;
      }
      var content = document.getElementById("content");
      const url = "https://" + window.location.host + "/files";
      var data = new FormData(content);
      data.append("from", user);
      data.append("to", person.textContent);
      data.append("quote", quote);
      data.append("type", "file");
      const fetchOpts = {
        method: content.method,
        body: data,
      };
      fetch(url, fetchOpts);
      evt.preventDefault();
      this.value="";
      quote = "";
      input.innerHTML = "<pre contenteditable='true'></pre>";
      return false;
    });
    
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
  
  document.getElementById("clear").addEventListener('click', function(evt) {
    let curl = "https://"+window.location.host+"/clear?user="+user;
    fetch(curl)
    .then((e) => {
      output.innerHTML = "";
    });
    return false;
  });
 
 
  record.addEventListener('click', function(evt) {
    evt.preventDefault();
    if (mediaRecorder.state == "inactive") {
      mediaRecorder.start();
	    this.style.background='orange';
    } else {
      mediaRecorder.stop();
      this.style.background='none';
    }
    return false;
  });
  
  document.getElementById('paste').addEventListener('click', function(evt) {
    evt.preventDefault();
    navigator.clipboard.readText()
  		.then((e) => {
  			var q = document.createElement("div");
  			q.className = "s-quoted";
  			var str = document.createElement("div");
  			str.className = "s-strip";
  			q.appendChild(str);
  			var qbx = document.createElement("div");
  			qbx.className = "s-qbox";
  			var spr = document.createElement("pre");
  			spr.textContent = e;
  			qbx.appendChild(spr);
  			q.appendChild(qbx);
  			input.appendChild(q);
  			var pre = document.createElement("pre");
  			pre.setAttribute('contenteditable', 'true'); 
  			pre.setAttribute('autofocus', 'true'); 

			input.appendChild(pre);
		});
		
  });
});
