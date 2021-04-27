<template>
<div class="modal-cover">
  <div class="modal-box">
    <slot></slot>
  </div>
</div>
</template>

<script>
export default {
    name: 'Modal'
}
</script>

<style>
.modal-cover {
    position: fixed;
    top: 0; left: 0;
    width: 100%; height: 100%;
    background-color: #0007;
    z-index: 99;
}
.modal-box {
    position: absolute;
    top: 50%; left: 50%;
    transform: translate(-50%, -50%);
    padding: 1em;
    background-color: var(--colBBkg);
}
</style>
