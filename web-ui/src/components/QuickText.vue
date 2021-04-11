<template>
<div id="quick">
  <label for="type">Quick type:
    <input id="type" v-model.lazy="type" placeholder="Press enter to send">
  </label>
  <span id="gap"></span>
  <label for="clip">Quick clip:
    <input id="clip" v-model.lazy="clip" placeholder="Press enter to send">
  </label>
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
