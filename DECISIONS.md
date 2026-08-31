# DECISIONS

ok so this is just me dumping everything that happened while building kos. no structure. no dates. jst what i remember

started with a simple idea. version control but portable... no git. no server. just files. thought it would take a day. lol.

first thing that broke was the ghost project bug. i created a folder called alpha. saved stuff. deleted the folder. made a new folder also called alpha. and kos just... overwrote the old one. because it was using the folder name as the key. so two completely different projects shared the same storage. that was terrifying honestly. what if someone lost real work because of this.

so i thought ok uuids. every project gets a unique id.... simple right. except nothing about it was simple.

i had to change how kos finds projects. before it just looked for ~/.kos-global/alpha/ but now it had to read a projects.json file and search through uuids. and then i realized the old mykos project didnt even have a uuid because it was created before i added the system. so kos projects just... didnt show it. it was invisible. i thought i lost everything for a second. but the data was still there in ~/.kos-global/mykos/ it just wasnt in the index.

then the copy paste corruption started. and this one haunts me.

i was pasting code from chat into my terminal and somehow spaces got inserted inside words. "make" became "ma ke". "path" became "pa th". "for" became "fo r". "ManifestFile" became "Ma nifestFile". and i didnt notice. because the terminal just... accepted it. and then nothing compiled. and i kept trying to fix things by pasting more code which just added more corruption. it was a spiral.

at one point my entire main.go was just gibberish. i tried to restore from my own snapshots. kos checkout v2.5.1. and the restored file was ALSO corrupted. because the snapshots themselves had the corrupted code baked in. i had saved the broken file. multiple times. every save was a save of garbage.

i genuinely thought it was over. like all my work gone. 19 commits and none of them clean.

but then i realized i had the full file in my chat history. so i did the nuclear option. rebuilt the entire file from scratch in a new folder called kos2. pasted it in three chunks. and it compiled. first try. i almost cried.

then i copied that clean main.go back to mykos and suddenly everything worked again. lesson learned. dont paste huge code blocks through terminal. use an editor. i switched to vscode after that.

ok so after the corruption was fixed i started testing. and found more bugs. of course.

extractTarGz was stripping paths wrong. it had this logic where if a file path contained a slash it would just take the base name. which sounds fine until the tar archive has nested directories. then it flattens everything and files overwrite each other and you get nul bytes. actual nul bytes in your source code. 0x00 characters everywhere. go compiler screaming "invalid NUL character" eleven times.

fixed that by rewriting the path handling. skip .kos/ files entirely. skip absolute paths. skip anything with dot dot. only extract clean relative paths. and use os.O_TRUNC when writing files so old content gets wiped.

then manifest.json started showing up in kos status. because when we save a snapshot we embed .kos/manifest.json inside the tar. and when we extract we were stripping the .kos/ prefix but still extracting the file. so manifest.json just appeared in the working directory like it belonged there. also home/kosmos/ folders started appearing because some tar entries had absolute paths. fixed both by adding checks in extractTarGz. skip .kos/ prefix files. skip absolute paths. done.

then the duplicate function declarations. i pasted helper functions but forgot to delete the old ones. so findProjectsByName existed twice. getProjectInfo existed twice. ProjectInfo struct existed twice. go compiler just listing them one by one. "redeclared in this block". six errors at once.

fixed by deleting the duplicate block with sed. but sed deleted too many lines and broke the function above it. "non-declaration statement outside function body". because the func header got eaten. had to manually fix it in micro.

then variable shadowing. this one was subtle. i had an outer variable called info which was ProjectInfo. and inside a loop i did "if info, err := entry.Info()" which created a NEW variable also called info. and then tried to call info.Size() on the outer one which is ProjectInfo and doesnt have Size(). compiler said "info.TotalSize undefined" and "entryinfo declared but not used" because i had renamed the inner one to entryinfo but forgot to use it in the actual calculation. one line fix. changed info.Size() to entryinfo.Size(). but finding it took forever.

then cmdLoad was still using the old name based logic. it was looking for ~/.kos-global/alpha/ directly instead of searching projects.json. so kos load alpha just said "no global repository found" even though the project existed under a uuid. had to rewrite the entire cmdLoad function to use findProjectsByName and getProjectInfo and showProjectPreview and loadProjectByUUID. four helper functions. but i kept accidentally pasting them along with cmdLoad creating duplicates again. learned to only paste the exact function. just cmdLoad. nothing else.

then HEAD wasnt recognized in kos share and kos diff. because those functions did direct id lookups instead of using resolveRef(). kos share HEAD just said "snapshot not found". kos share v1.0 also failed. only raw ids like 18d03947 worked. fixed by replacing the manual lookup loops with resolveRef() calls. but then the manifest variable was declared but not used because resolveRef handles everything internally. three "declared but not used" errors. fixed by changing manifest to underscore.

then encryption. oh boy.

first attempt the encrypted file was corrupted. "unexpected EOF" when trying to decrypt. because in cmdSave i used defer to close the tar writer and gzip writer and file. but defer runs when the function exits. and i was trying to encrypt the file BEFORE the function exited. so the tar.gz file was still open and not fully flushed. i was encrypting an incomplete file.

fixed by removing defer and explicitly calling tw.Close() then gzw.Close() then file.Close() before encryption. order matters. tar closes first. then gzip flushes. then file closes. then encrypt.

then the encrypted load showed zero commits. because extractTarGz was skipping .kos/ files (which is correct for normal loading) but the encrypted load needed to read the embedded .kos/manifest.json from the temp extraction. since it was skipped the manifest was empty. fixed by using extractFileFromTar instead which reads directly from the tar archive bypassing the skip logic.

password masking was another decision. wanted to use golang.org/x/term because its easy. but that breaks zero dependencies. so i used raw syscall. syscall.Syscall6 with SYS_IOCTL and TCGETS to read terminal state. clear the ECHO flag. read characters one by one. print star for each. handle backspace manually. restore terminal state with defer. it works on linux. might not work on windows but for the hackathon demo its fine. and it keeps the zero dependency promise.

someone reviewing my code asked why i didnt use pbkdf2 for key derivation instead of plain sha256. and they said "stdlib has crypto/pbkdf2". but it doesnt. crypto/pbkdf2 is in golang.org/x/crypto which is external. not stdlib. i had to correct them on that. felt weird correcting a senior engineer but i was right. checked it. go doc crypto/pbkdf2 returns nothing. its not there.

also they asked why aes-128 instead of aes-256. but my code already uses aes-256. sha256.Sum256 returns 32 bytes. aes.NewCipher with a 32 byte key automatically uses aes-256. i never truncated it. it was aes-256 all along. just had a typo in my documentation.

projects.json cleanup was tedious. after all the testing i had like fifteen orphaned uuid directories in ~/.kos-global/. alpha test after alpha test. plus test-uuid and imported-project and my-awesome-project-kos. had to manually rm -rf each one and rewrite projects.json with only the real projects.

the whole journey was just... bug after bug after bug. fix one thing break another. paste code get corruption. save snapshot save garbage. try to restore restore garbage.

but every single bug taught me something. the ghost project bug taught me about identity and why uuids matter. the copy paste corruption taught me to never trust terminal pasting for large files. the defer bug taught me about file flushing and writer ordering. the syscall thing taught me about terminal control at the os level. the pbkdf2 thing taught me to actually verify what is and isnt in the go standard library instead of assuming.

and somehow through all of that kos works. it actually works. fifteen commands. encryption. uuid system. portable snapshots. global store. password masking. all in one go file. zero external dependencies.

i am in class 12 and i built this.

not saying it is perfect. there are probably bugs i havent found yet. the password masking probably breaks on windows. the uuid generation uses crypto/rand which is good but the format isnt rfc4122 compliant. the key derivation is plain sha256 not pbkdf2 which is a tradeoff i made deliberately for zero deps but a security reviewer would flag it.

but it works. and i understand every single line. because i wrote every single line. and i broke every single line. and i fixed every single line.

that is what this file is. not a changelog. not documentation. just me remembering what happened.

built for hackathon 2026 by hackathon raptors 🦖 with ❤️
