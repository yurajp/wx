
window.users = [];

const types = {
  'mp3': 'audio',
  'wav': 'audio',
  'ogg': 'audio',
  'txt': 'text',
  'pdf': 'text',
  'docx': 'text',
  'png': 'image',
  'jpg': 'image',
  'mp4': 'video',
  'webm': 'video',
  'mpeg': 'video',
  'go': 'code',
  'html': 'code',
  'css': 'code',
  'kt': 'code',
  'js': 'code',
  'py': 'code',
  'zip': 'archive',
  'rar': 'archive'
};



window.addEventListener("load", function(evt) {
    var user = document.getElementById("username").textContent;
    var output = document.getElementById("output");
    var usermenu = document.getElementById("usermenu");
    var person = document.getElementById("person");
    var input = document.getElementById("input");
    var select = document.querySelectorAll("sel");
    var home = window.location.host;
    var sound = new Audio('static/elegant.mp3');
    
    var ws;
    
  
    var print = function(message) {
      if (message.startsWith('USERS@')) {
        let us = JSON.parse(message.slice(6));
        window.users = ["+All"];
        window.users.push( ...us);

        return false;
      }
      if (message == 'CLOSED') {
        ws = null;
        return false;
      }
      if (message.startsWith('ERROR')) {
        ws.write(message);
        return false;
      }
      if (message.startsWith("FILE@")) {
        let fname = message.replace("FILE@", "");
        
        var bx = document.createElement("div");
        bx.className = "flink";
        var lnk = "https://"+home+"/files/"+fname;
        var ext = fname.split('.')[1];
        var med = types[ext];
        if (!med) {
          med = 'other';
        }
        var timg = document.createElement("img");
        var ipath = "static/types/" + med + ".png";
        timg.setAttribute("src", ipath);
        timg.setAttribute("target", "");
        var trg = "_blank";
        var a = document.createElement("a");
        a.setAttribute("href", lnk);
        a.setAttribute("target", trg);
        a.appendChild(timg);
        var fnm = document.createTextNode(fname);
        a.appendChild(fnm);
        var da = document.createElement("div");
        da.className = "ftxt";
        da.appendChild(a);
        bx.appendChild(da);
        
        var b = document.createElement("a");
        b.setAttribute("href", lnk);
        b.setAttribute("target", trg);
        b.setAttribute('download', "download");
        
        var bt = document.createElement("img");
        bt.setAttribute("src", "static/download_button.png");
        bt.caption = "";
        b.appendChild(bt);
        
        var db = document.createElement("div");
        db.className = "fbt";
        db.appendChild(b);
        bx.appendChild(db);
        output.appendChild(bx);
        
        return false;
      }
      
      var spl = message.split("\n");
      var box = document.createElement("div");
      box.className = "inbox";
      if (spl.length > 1) {
        var sel = document.createElement("span");
        sel.className = "sel";
        sel.textContent = "✫";
        box.appendChild(sel);
       
        var cur = document.createElement("span");
        cur.className = "cursiv";
        cur.textContent=spl[0];
        box.appendChild(cur);
        var txt = document.createElement("div");
        txt.className = "txt";
        txt.innerHTML = "<pre>" + spl.slice(1).join("\n") + "</pre>";
        sel.addEventListener('click', (e)=>{
          if (sel.style.color == "teal") {
            return false;
          }  
          let text = "<div class='inbox'>" + sel.parentNode.innerHTML + "</div>";
          var js = JSON.stringify({"from":user, "to": 'STORE', "text": text});
          ws.send(js);
          sel.style.color = 'teal';
          return false;
        });
        
        box.appendChild(txt);
      } else {
        var serv = document.createElement("p");
        serv.textContent = message;
        serv.className = "srv";
        box.appendChild(serv);
      }
      output.appendChild(box);
      output.scroll(0, output.scrollHeight);
    };

    document.getElementById("avatar").setAttribute("src", "static/avatars/" + user + ".jpg");

    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        
        ws = new WebSocket("wss://"+window.location.host+"/translator");
        ws.onopen = function(evt) {
            print("Hi, "+user+"!");
        document.getElementById("chat").style.display='flex';
            
        };
        ws.onclose = function(evt) {
            print("CLOSED");
    
          window.location.href= "https://" + home + "/";
        };
        ws.onmessage = function(evt) {
            print(evt.data);
            sound.play();
        };
        ws.onerror = function(evt) {
            print("ERROR" + evt.data);
        };
        document.getElementById('open').style.display='none';
        document.getElementById('close').style.display='inline-block';
        
        return false;
    };

    
    document.getElementById("send").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        let from = user;
        let to = person.textContent;
        let text = input.value;
        var js = JSON.stringify({"from": from, "to": to, "text": text});
        ws.send(js);
        person.textContent = "all";
        input.value = "";
        return false;
    };

    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.send("CLOSED");
        ws.close(1001, "going away");
        
        return false;
    };
    
    person.onclick = function(evt) {
      if (!window.users) {
        return false;
      }
      if (usermenu.style.display == "block") {
        usermenu.style.display = 'none';
        return false;
      }
      usermenu.innerHTML='';
      var ul = document.createElement("ul");
      window.users.forEach(function(item) {
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
      const fetchOpts = {
        method: content.method,
        body: data,
      };
      fetch(url, fetchOpts);
      
      evt.preventDefault();
      this.value="";
      return false;
    });
    
});

