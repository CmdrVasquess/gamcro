<template>
<div id="quick">
  <label for="type" class="before">Quick type</label>
  <input id="type" v-model.lazy="type"
         placeholder="Text to type on target">
  <span id="gap"></span>
  <label for="clip" class="before">Quick clip</label>
  <input id="clip" v-model.lazy="clip"
         placeholder="Text to send to target clipboard">
</div>
</template>

<script>
export default {
    name: 'QuickText',
    props: {typeMsg: Function, clipMsg: Function},
    data() {
        return {
            type: "",
            clip: ""
        }
    },
    watch: {
        type(n) {
            if (n != "") this.typeMsg(n);
            this.type = "";
        },
        clip(n) {
            if (n != "") this.clipMsg(n);
            this.clip = "";
        }
    }
}
</script>

<style>
#gap { display: inline-block; width: 2em; }  
</style>
