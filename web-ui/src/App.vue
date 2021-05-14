<template>
<transition name="fade">
  <div v-if="status" id="stat" title="Status message. Click to close."
       @click="status=''">{{status}}</div>
</transition>
<h1><a href="https://github.com/CmdrVasquess/gamcro/wiki"
  target="gamcro-wiki"><img src="logo.png" height="38" alt="G"></a>amcro
  • Text Panel</h1>
<main>
  <div id="top">
    <QuickText :typeMsg="typeMsg" :clipMsg="clipMsg"/>
    <div>
      <button @click="addMsg()">New Text</button>
      <button v-if="api('ClipGetAPI')" title="New text from remote clipboard"
              @click="getClip()">Get remote Clip</button>
    </div>
  </div>
  <transition-group name="msgs">
    <Message v-for="(msg, idx) in msgs" :key="msg.key" :index="idx"
             :typeMsg="typeMsg" :clipMsg="clipMsg" :delMsg="delMsg"
             v-model="msgs[idx].text"/>
  </transition-group>
</main>
<span class="menu" v-if="menu" @click="menu=false">✕</span>
<span class="menu" v-else @click="menu=true">≡</span>
<transition name="menu">
  <aside v-if="menu" class="menu">
    <span style="display:inline-block;padding-bottom:.5em">v{{version}}</span>
    <div @click="menu=false;modal='import'" class="button">↴ Import Texts</div>
    <div @click="menu=false;modal='export'" class="button">Export Texts ↱</div>
    <div v-if="api('SaveTexts')"
         @click="menu=false;modal='save'" class="button">🖫 Save Texts</div>
    <div v-if="!cfg.MultiClient" @click="menu=false;disconnect()" class="button"
         title="Allow to connect from anoter machine">Disconnect</div>
  </aside>
</transition>
<transition name="fade">
  <Modal id="export" v-if="modal=='export'">
    <h1>Export texts</h1>
    <textarea v-model="exportText" readonly></textarea>
    <button @click="modal=''">Close</button>
  </Modal>
</transition>
<transition name="fade">
  <Modal id="export" v-if="modal=='import'">
    <h1>Import texts</h1>
    <textarea v-model="impTexts"></textarea>
    <button @click="modal='';importText()"
            :disabled="!impValid">Import</button>
    <button @click="modal=''">Close</button>
  </Modal>
</transition>
<transition name="fade">
  <Modal id="save" v-if="modal=='save'">
    <h1>Save texts in Gamcro</h1>
    <div>
      <label>As</label> <input type="text" v-model="saveName">
    </div>
    <button @click="modal='';saveTexts(saveName);saveName=''"
            :disabled="saveName.length==0">Save</button>
    <button @click="modal=''">Cancel</button>
  </Modal>
</transition>
<footer>
  <span title="If you like Gamcro, I would be happy to receive your appreciation.">Buy me a coffee: </span>
  <a href="https://flattr.com/@CmdrVasquess" target="_blank">
    <img src="flattr-badge.png" alt="flattr" id="flattr"/>
  </a> •
  <a href="https://liberapay.com/CmdrVasquess/donate" target="_blank">
    <img src="lp-badge.png" alt="LiberaPay" id="liberapay"/>
  </a>
</footer>
</template>

<script>
import QuickText from './components/QuickText.vue'
import Message from './components/Message.vue'
import Modal from './components/Modal.vue'

export default {
    name: 'App',
    components: {
        QuickText,
        Message,
        Modal
    },
    data() {
        return {
            version: process.env.VUE_APP_VERSION,
            status: "",
            msgs: [],
            menu: false,
            imexpdlg: false,
            msgseq: 0,
            modal: "",
            impTexts: "",
            saveName: "",
            cfg: {
                "Version": "?.?.?",
                "APIs": ["ClipGetAPI","SaveTexts"],
                "MultiClient": false
            }
        }
    },
    computed: {
        exportText() {
            let txts = [];
            for (let m of this.msgs) {
                txts.push(m.text);
            }
            return JSON.stringify(txts, null, 2);
        },
        impValid() {
            try {
                let msgs = JSON.parse(this.impTexts);
                return Array.isArray(msgs);
            } catch(x) {
                return false;
            }
         }
    },
    methods: {
        addMsg() {
            this.msgseq++;
            let msg = {key: this.msgseq, text: ""};
            this.msgs.unshift(msg);
        },
        delMsg(i) {
            if (this.msgs.length > 1) {
                this.msgs.splice(i, 1);
            } else {
                this.msgs[0].text = "";
            }
        },
        typeMsg(txt) { this.sendMsg("/keyboard/type", txt); },
        clipMsg(txt) { this.sendMsg("/clip", txt); },
        sendMsg(op, txt) {
            console.log("send", op, txt);
            this.status = "";
            let init = {
                method: "POST",
                headers: {'Content-Type': "text/plain"},
                body: txt
            };
            fetch(new Request(op, init))
                .then(resp => {
                    if (!resp.ok) {
                        this.status = resp.statusText;
                    }
                    this.qtype = "";
                    this.qclip = "";
                });
        },
        getClip() {
            this.modal = 'remote-clip';
            fetch('/clip')
                .then(resp => {
                    if (resp.ok)
                        return resp.text();
                    console.log(resp.status, resp.statusText);
                    this.status="An error occured while getting remote clipboard";
                    return "";
                })
                .then(txt => {
                    if (txt.length==0) return;
                    this.msgseq++;
                    let msg = {key: this.msgseq, text: txt};
                    this.msgs.unshift(msg);
                })
                .catch(x => this.status=x);
        },
        importText() {
            try {
                let txts = JSON.parse(this.impTexts);
                let msgs = []
                for (let i in txts) {
                    msgs.push({key: i, text: txts[i]});
                }
                this.msgs = msgs;
                this.msgseq = this.msgs.length;
            } catch(x) {
                console.log("import texts: ", x);
            }
        },
        saveTexts(name) {
            let init = {
                method: "POST",
                headers: {'Content-Type': "application/json"},
                body: this.exportText
            };
            fetch(new Request('/texts/'+name, init))
                .then(resp => {
                    if (!resp.ok) {
                        this.status = resp.statusText;
                    }
                });
        },
        disconnect() {
            fetch("/client/release")
                .then(resp => {
                    console.log("disconnect:", resp.status);
                });
        },
        api(name) {
            for (let n of this.cfg.APIs) {
                if (n == name) return true;
            }
            return false;
        }
    },
    mounted() {
        if (localStorage.msgs) {
            try {
                let msgs = JSON.parse(localStorage.msgs);
                if (msgs.length > 0) {
                    if (typeof msgs[0] == 'string' || msgs[0] instanceof String) {
                        for (let i in msgs) {
                            this.msgs.push({key: i, text: msgs[i]});
                        }
                    } else {
                        for (let i in msgs) {
                            msgs[i].key = i;
                        }
                        this.msgs = msgs;
                    }
                    this.msgseq = this.msgs.length;
                }
            } catch(x) {
                console.log("load msgs from local storage: ", x);
            }
        }
        if (this.msgs.length == 0) {
            this.msgs = [{key: this.msgseq, text: ""}];
            this.msgseq = 0;
        }
        fetch("/config")
            .then(resp => resp.json())
            .then(conf => {
                console.log(conf);
                this.cfg = conf;
            })
            .catch(() => console.log("failed to fetch config"));
    },
    watch: {
        msgs: {
            handler() {
                localStorage.msgs = JSON.stringify(this.msgs);
            },
            deep: true
        }
    }
}
</script>

<style>
:root {
    --colBkg: #203244;
    --colBkgT: #203244cc;
    --colFgr: #F0B80F;
    --colBBkg: #5F83A7;
}
body {
    background-color: var(--colBkg);
    color: var(--colFgr);
}
button {
    background-color: var(--colBBkg);
    color: var(--colFgr);
    font-weight: bold;
    font-size: 102%;
    border: 2px solid var(--colBBkg);
    border-radius: .3em;
    box-shadow: 0 0 .4em 1px black;
    margin: .4em;
    padding: .4em 1em;
}
button:focus { border: 2px solid #A0BAD5; }
button:hover { color: #F7D87B; }
button:active { box-shadow: none; }
button:disabled {
    box-shadow: none;
    color: #A0BAD5;
    cursor: not-allowed;
}
a:any-link {
    color: var(--colFgr);
    text-decoration: none;
}
label::after { content: ':'; }
label.before {
    font-weight: bold;
    margin-right: .3em;
}
input {
    background-color: var(--colBkg);
    color: var(--colFgr);
    font-size: 105%;
    border: none;
    border: 2px solid var(--colBkg);
    border-bottom: 2px solid var(--colBBkg);
    padding: .2em .5em;
    padding-bottom: .1em;
    margin-bottom: .1em;
}
input:focus { border: 2px solid #A0BAD5; }
textarea {
    background-color: black;
    color: var(--colFgr);
    font-size: 130%;
    border: 2px solid black;
    padding: .2em .5em;
}
textarea:focus { border: 2px solid #A0BAD5; }
#app {
    font-family: Avenir, Helvetica, Arial, sans-serif;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    text-align: center;
}
span.menu {
    position: fixed;
    right: 0; top: 0;
    z-index: 3;
    font-size: 160%;
    padding: .1em 0;
    width: 1.6em;
    text-align: center;
    cursor: pointer;
    background-color: var(--colBkgT);
    border-radius: 0 0 0 .3em;
}
aside.menu {
    position: fixed;
    right: 0; top: 0;
    z-index: 2;
    padding: .5em 1.5em;
    padding-top: 1em;
    height: 100vh;
    background-color: black;
    box-shadow: 0 0 .4em 1px black;
}
.menu-enter-active {
    transition: all 0.3s ease;
}

.menu-leave-active {
    transition: all 0.3s ease;
}
.menu-enter-from,
.menu-leave-to {
    transform: translateX(40px);
    opacity: 0;
}
aside {
    color: var(--colBBkg);
    text-align: left;
}
aside > div {
    font-weight: bold;
    padding: .3em .7em;
    margin: .2em;
}
aside > div.button {
    cursor: pointer;
}
aside > div.button:hover {
    color: var(--colFgr);
}
h1 { margin: .5em 0; }
img[src="logo.png"] { padding-right: .1em; }
main {
    max-width: 60em;
    margin: auto;
    padding-bottom: 1.7em;
}
#top {
    display: flex;
    flex-flow: row wrap;
    justify-content: space-between;
    align-content: space-between;
    align-items: baseline;
    background-color: var(--colBkg);
    position: sticky;
    top: 0;
    padding: 0.5em 0;
}
#stat {
    background-color: black;
    color: red;
    font-weight: bold;
    padding: .7em 0;
    margin: .5em 0;
    cursor: pointer;
}
.message { margin-top: 1em; }
.fade-enter-active,
.fade-leave-active {
    transition: opacity 0.4s ease;
}
.fade-enter-from,
.fade-leave-to {
    opacity: 0;
}
.message {
    transition: all 0.4s ease;
}
.msgs-enter-from,
.msgs-leave-to {
    opacity: 0;
    margin: 0;
    transform: scale(1, 0) translateX(-80px);
}
.msgs-leave-active {
    height: 0;
}
div.modal-box h1 {
    margin-top: 0;
    font-size: 150%;
}
div.modal-box textarea {
    width: 60vw;
    height: 60vh;
}
footer {
    position: fixed;
    left: 0; bottom: 0; width: 100%;
    background-color: #443820;
    color: #BD910C;
    text-align: left; padding: .1em 1em;
    z-index: 1;
}
footer img {
    height: .8em;
    vertical-align: baseline;
}
footer #liberapay {
    height: 1.1em;
    vertical-align: baseline;
}
</style>
