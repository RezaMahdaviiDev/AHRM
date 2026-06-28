(function () {
  'use strict';

  var es = new EventSource('/events');

  es.onmessage = function (e) {
    toast(e.data);
    beep();
  };

  es.onerror = function () {
    // browser auto-reconnects; no action needed
  };

  function beep() {
    try {
      var AC = window.AudioContext || window.webkitAudioContext;
      if (!AC) return;
      var ctx = new AC();
      var osc = ctx.createOscillator();
      var gain = ctx.createGain();
      osc.connect(gain);
      gain.connect(ctx.destination);
      osc.type = 'sine';
      osc.frequency.value = 880;
      gain.gain.setValueAtTime(0.35, ctx.currentTime);
      gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.6);
      osc.start(ctx.currentTime);
      osc.stop(ctx.currentTime + 0.6);
    } catch (_) {}
  }

  function toast(msg) {
    var div = document.createElement('div');
    div.setAttribute('style', [
      'position:fixed',
      'top:1.25rem',
      'left:50%',
      'transform:translateX(-50%)',
      'background:#1e40af',
      'color:#fff',
      'padding:0.75rem 1.5rem',
      'border-radius:10px',
      'z-index:9999',
      'max-width:90vw',
      'text-align:center',
      'direction:rtl',
      'font-family:Tahoma,sans-serif',
      'font-size:0.95rem',
      'box-shadow:0 4px 16px rgba(0,0,0,0.4)',
      'transition:opacity 0.4s',
      'white-space:pre-line'
    ].join(';'));
    div.textContent = msg;
    document.body.appendChild(div);
    setTimeout(function () {
      div.style.opacity = '0';
      setTimeout(function () { div.remove(); }, 400);
    }, 6000);
  }
})();
