import React, { useState, useRef, useEffect } from 'react';
import SimplePeer from 'simple-peer';
import io from 'socket.io-client';

const socket = new WebSocket('ws://localhost:8080/ws');

const App = () => {
  const [file, setFile] = useState(null);
  const [peer, setPeer] = useState(null);
  const fileInputRef = useRef(null);

  useEffect(() => {
    socket.onmessage = (message) => {
      const { type, payload } = JSON.parse(message.data);

      if (type === 'offer') {
        const peer = new SimplePeer({
          initiator: false,
          trickle: false
        });

        peer.on('signal', data => {
          socket.send(JSON.stringify({ type: 'answer', payload: JSON.stringify(data) }));
        });

        peer.on('data', (data) => {
          const receivedFile = new Blob([data]);
          const downloadLink = document.createElement('a');
          downloadLink.href = URL.createObjectURL(receivedFile);
          downloadLink.download = 'received_file';
          document.body.appendChild(downloadLink);
          downloadLink.click();
          document.body.removeChild(downloadLink);
        });

        peer.signal(JSON.parse(payload));
        setPeer(peer);
      } else if (type === 'answer') {
        peer.signal(JSON.parse(payload));
      }
    };

    const newPeer = new SimplePeer({
      initiator: true,
      trickle: false
    });

    newPeer.on('signal', data => {
      socket.send(JSON.stringify({ type: 'offer', payload: JSON.stringify(data) }));
    });

    setPeer(newPeer);
  }, []);

  const handleFileChange = (event) => {
    setFile(event.target.files[0]);
  };

  const handleSendFile = () => {
    const reader = new FileReader();
    reader.onload = () => {
      peer.send(reader.result);
    };
    reader.readAsArrayBuffer(file);
  };

  return (
      <div>
        <h2>WebRTC File Transfer</h2>
        <input type="file" ref={fileInputRef} onChange={handleFileChange} />
        <button onClick={handleSendFile} disabled={!file}>Send File</button>
      </div>
  );
};

export default App;
