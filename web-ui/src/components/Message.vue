<template>
<div class="message">
  <textarea placeholder="Enter text to send"
            :value="modelValue"
            @input="$emit('update:modelValue', $event.target.value)"></textarea>
  <button :disabled="modelValue.length==0"
          @click="typeMsg(modelValue)"
          title="Type text on the target computer">Type</button>
  <button :disabled="modelValue.length==0"
          @click="clipMsg(modelValue)"
          title="Send text to clipboard of the target computer">Clip</button>
  <button id="del" @click="delMsg(index)">Delete</button>
</div>
</template>

<script>
export default {
    name: 'Message',
    props: {
        modelValue: String,
        index: Number,
        typeMsg: Function, clipMsg: Function, delMsg: Function
    },
    emits: ['update:modelValue']
}
</script>

<style>
.message {
    display: grid;
    grid-template-columns: 2fr 2fr 1fr 1fr;
}
.message > textarea { grid-column: 1/5; }
#del { grid-column: 4/5; }
</style>
