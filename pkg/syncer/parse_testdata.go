package syncer

type imageTestCase struct {
	link    string
	content string
	md      string
}

var mdLinkCases = []imageTestCase{
	{
		"./subfolder/note.md",
		"[note title](./subfolder/note.md)",
		"[note title](./subfolder/note.md)",
	},
	{ // absolute path
		"/home/user/notes/my_note.md",
		`[my_note](/home/user/notes/my_note.md)`,
		`[my_note](/home/user/notes/my_note.md)`,
	},
	{
		"../note.md",
		`[my_note](../note.md)`,
		`[my_note](../note.md)`,
	},
	{ // nested parentheses
		"./(notes)/note.md",
		`[my_note](./(notes)/note.md)`,
		`[my_note](./(notes)/note.md)`,
	},
	{ // ref
		"note.md",
		"[note1Id]: note.md",
		`[note name][note1Id]
[note1Id]: note.md`,
	},
}

var mdImageCases = []imageTestCase{
	{
		"./assets/subfolder/image.png",
		"[alt text](./assets/subfolder/image.png)",
		"![alt text](./assets/subfolder/image.png)",
	},
	{
		"no-alt-text.png",
		`[alt text](no-alt-text.png)`,
		`![alt text](no-alt-text.png)`,
	},
	{
		"assets/img2.jpeg",
		`[alt text](assets/img2.jpeg "image title")`,
		`![alt text](assets/img2.jpeg "image title")`,
	},
	{
		"img3.gif",
		`[alt text](img3.gif "title")`,
		`![alt text](img3.gif "title")`,
	},
	{ // absolute path
		"/home/user/notes/assets 2/name with spaces.jpg",
		`[alt text](/home/user/notes/assets 2/name with spaces.jpg)`,
		`![alt text](/home/user/notes/assets 2/name with spaces.jpg)`,
	},
	{
		"../assets/img4.svg",
		`[alt text](../assets/img4.svg "title")`,
		`![alt text](../assets/img4.svg "title")`,
	},
	{
		"../../outside_dir/img5.svg",
		`[alt text](../../outside_dir/img5.svg)`,
		`![alt text](../../outside_dir/img5.svg)`,
	},
	{
		"./non_latin/изображение.svg",
		`[alt text](./non_latin/изображение.svg)`,
		`![alt text](./non_latin/изображение.svg)`,
	},
	{
		`./"img".png`,
		`[alt text](./"img".png)`,
		`![alt text](./"img".png)`,
	},
	// nested parentheses
	{
		"./(assets)/img.png",
		`[alt text](./(assets)/img.png)`,
		`![alt text](./(assets)/img.png)`,
	},
	// escaped parentheses
	{
		`./(assets)/img).png`,
		`[alt text](./(assets)/img\).png)`,
		`![alt text](./(assets)/img\).png)`,
	},
	// nested brackets
	{
		"./(assets)/img.png",
		`[alt [text]](./(assets)/img.png)`,
		`![alt [text]](./(assets)/img.png)`,
	},
	{ // encoded
		"./%D0%B8/%D1%81%D1%85%D0%B5%D0%BC%D0%B0.svg",
		`[alt text](./%D0%B8/%D1%81%D1%85%D0%B5%D0%BC%D0%B0.svg)`,
		`![alt text](./%D0%B8/%D1%81%D1%85%D0%B5%D0%BC%D0%B0.svg)`,
	},
	// ref
	{
		"assets/ref_image.png",
		"[imgid1]: assets/ref_image.png",
		`![alt text][imgid1]
[imgid1]: assets/ref_image.png "title"`,
	},
	// video
	{
		"./assets/img6.png",
		"[video](./assets/img6.png)",
		"[![video](./assets/img6.png)](https://youtube.com)",
	},
	// HTML
	{
		"assets/img7.webp",
		`<img src="assets/img7.webp" alt="alt text" style="zoom:50%;" />`,
		`<img src="assets/img7.webp" alt="alt text" style="zoom:50%;" />`,
	},
	{
		"../assets/img8.png",
		`<img src = "../assets/img8.png" alt="alt text" />`,
		`<img src = "../assets/img8.png" alt="alt text" />`,
	},
	{
		"img9.png",
		`<img src=img9.png alt="alt text" />`,
		`<img src=img9.png alt="alt text" />`,
	},
	{
		`images/"quotes".png`,
		`<img src=images/"quotes".png  />`,
		`<img src=images/"quotes".png  />`,
	},
	{
		`images/"quotes2".png`,
		`<img src='images/"quotes2".png' alt="alt text" />`,
		`<img src='images/"quotes2".png' alt="alt text" />`,
	},
}
