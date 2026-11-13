Bring go-gitutil's file_status.go, ignored_files.go, info_store.go and repo.go up to gomion's gommod/gitutils versions, which are the newer copies, so the two packages agree before gomion deletes its copy and imports this one.

For each file: diff go-gitutil's against gomion's, apply gomion's changes, and keep anything go-gitutil has that gomion lacks (worktree.go is go-gitutil's alone and is untouched). Where repo.go's drift meets E-2600's reshaping of Repo, do this first on the small diff; E-2600 then reshapes one agreed file. Build and vet; the existing diff parser test still passes.
