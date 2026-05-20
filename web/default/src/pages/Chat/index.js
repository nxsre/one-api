import React from 'react';

const Chat = () => {
  const chatLink = localStorage.getItem('chat_link');

  return (
    <iframe src={chatLink} title='chat' />
  );
};


export default Chat;
