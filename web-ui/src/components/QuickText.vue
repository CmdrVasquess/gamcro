<template>
<div id="quick">
  <input id="type" v-model.lazy="type" placeholder="Quick type"
         title="Text to type on target computer">
  <input id="clip" v-model.lazy="clip" placeholder="Quick clip"
         title="Text to send to target computer's clipboard">
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
#quick {
    display: flex;
    flex-wrap: wrap;
    gap: 20px;
}
</style>
