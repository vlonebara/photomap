<script setup lang="ts">
defineProps<{ message?: string }>()
</script>

<template>
  <span v-if="message" class="panel-error" role="alert">{{ message }}</span>
</template>
