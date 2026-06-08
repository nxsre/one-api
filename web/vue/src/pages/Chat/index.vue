<template>
  <iframe :src="chatLink" title="chat" />
</template>

<script setup>
const chatLink = localStorage.getItem('chat_link');
</script>
