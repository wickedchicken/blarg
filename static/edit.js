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

sendMessage = function(path, opt_param) {
  var session_key = document.getElementById("session").innerHTML;
  path += '?g=' + session_key;
  //if (opt_param) {
  //  path += '&' + opt_param;
  //}
  var xhr = new XMLHttpRequest();
  xhr.open('POST', path, true);
  if (opt_param) {
    xhr.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
    xhr.send(opt_param);
  } else {
    xhr.send();
  }
};

sendObject = function(path, paramname, obj){
  var str = JSON.stringify(obj);
  sendMessage(path, escape(paramname) + "=" + escape(str));
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
  sendObject('/admin/render/', 'data', data);
  update_status("processing...", "#AAAAAA");
}

save_post = function(){
  var data = {}
  data['data'] = get_elem("inputbox");
  data['title'] = get_elem("title");
  data['labels'] = get_elem("labels");
  sendObject('/admin/post/', 'data', data);
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
