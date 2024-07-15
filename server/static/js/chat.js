  function deleteMessage(sid) {
    let user = document.getElementById("uname").textContent;
    let urdl = 'https://'+window.location.host+"/delete?sid="+sid+"&user="+user;
    let msg = document.getElementById(sid);
    
    fetch(urdl)
    .then((res) => res.ok)
    .then((ok) => {
      if (ok) {
        msg.style.display='none';
      }
    });
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
  q.style.background = '#E5A015';
    
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
        fd.append('voice', voiceBlob);

        fetch(url, {
	        method: 'POST',
	        body: fd})
        .then((e) => {
	        voice = [];
        });
      });
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
          handleMessage(evt.data);
          sound.play();
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
        let text = input.value;
        var js = JSON.stringify({from:user,to:to,type:"text",data:text,quote:quote});
        ws.send(js);
        person.textContent = "All";
        input.value = "";
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
      const data = new FormData(content);
      data.append("from", user);
      data.append("to", person.textContent);
      data.append("type", "file");
      const fetchOpts = {
        method: content.method,
        body: data,
      };
      fetch(url, fetchOpts);
      evt.preventDefault();
      this.value="";
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
});


