// from http://unscriptable.com/2009/03/20/debouncing-javascript-methods/
Function.prototype.debounce = function (threshold, execAsap) {
  var func = this, timeout;

  return function debounced () {
    var obj = this, args = arguments;
    function delayed () {
      if (!execAsap)
        func.apply(obj, args);
      timeout = null; 
    };

    if (timeout)
      clearTimeout(timeout);
    else if (execAsap)
      func.apply(obj, args);

    timeout = setTimeout(delayed, threshold || 100); 
  };
}

update = function(text){
  document.getElementById("display").innerHTML = text;
};

update_status = function(text, color){
  document.getElementById("status").innerHTML = text;
  document.getElementById("status").style.color = color;
}

get_elem = function(elem){
  return document.getElementById(elem).value;
}

sendMessage = function(path, opt_params, appendg) {
  var session_key = document.getElementById("session").innerHTML;
  if(appendg){
    path += '?g=' + session_key;
  }
  //if (opt_param) {
  //  path += '&' + opt_param;
  //}
  var error = function(status){
    update_status("error connecting to server: " + status, "#AA0000");
  }
  var xhr = new XMLHttpRequest();
  xhr.open('POST', path, true);
  xhr.onreadystatechange = function(){ 
    if ( xhr.readyState == 4 ) { 
      if ( xhr.status == 200 ) { 
        //success(xhr.responseText); 
      } else { 
        error(xhr.status); 
      } 
    } 
  }
  xhr.onerror = function () { 
    error(xhr.status); 
  }
  if (opt_params) {
    var multipart = "";
    var boundary=Math.random().toString().substr(2);
    xhr.setRequestHeader("content-type",
                "multipart/form-data; charset=utf-8; boundary=" + boundary);
    for(var key in opt_params){
      multipart += "--" + boundary
                 + "\r\nContent-Disposition: form-data; name=\"" + key +"\""
                 + "; filename=\"temp.txt\""
                 + "\r\nContent-type: application/octet-stream"
                 + "\r\n\r\n" + opt_params[key] + "\r\n";
    }
    multipart += "--"+boundary+"--\r\n"; 
    xhr.send(multipart);
  } else {
    xhr.send();
  }
};

sendObject = function(path, paramname, obj, sendg){
  var str = JSON.stringify(obj);
  var hash = {}
  hash[paramname] = str;
  sendMessage(path, hash, sendg);
}

onOpened = function() {
  connected = true;
  update_status("connected to server", "#00AA00");
};

onMessage = function(input){
  var message = JSON.parse(input.data);
  if('markdown' in message) {
    update(message['markdown']);
    update_status("updated", "#00AA00");
  }
  if('status' in message) {
    if('color' in message) {
      update_status(message['status'], message['color']);
    } else {
      update_status(message['status'], "#AAAAAA");
    }
  }
};

onError = function(){
  update_status("error connecting to server!", "#AA0000");
}

onClose = function(){
  update_status("connection closed! please reload page.", "#AA0000");
}

submit_text = function(){
  var data = {}
  data['data'] = get_elem("inputbox");
  sendObject('/admin/render/', 'data', data, true);
  update_status("processing...", "#AAAAAA");
}

save_post = function(){
  var data = {}
  data['data'] = get_elem("inputbox");
  data['title'] = get_elem("title");
  data['labels'] = get_elem("labels");
  url = document.getElementById('uploadurl').innerHTML;
  sendObject(url, 'data', data, false);
  update_status("saving...", "#AAAAAA");
}

initialize = function(){
  token = document.getElementById("token").innerHTML;
  channel = new goog.appengine.Channel(token);
  socket = channel.open();
  socket.onopen = onOpened;
  socket.onmessage = onMessage;
  socket.onerror = onError;
  socket.onclose = onClose;

  debouncedTextSend = submit_text.debounce(500, false);
  document.getElementById('savelink').onclick = function() { 
    save_post();
  }
}

window.addEventListener('DOMContentLoaded', initialize, false);
