
<img src="https://www.devteam.space/wp-content/uploads/2017/03/gopher_head-min.png"/>

# Projecto

Projecto opens up your project folder in your specified editors.

# Installation

Make sure golang is installed on your machine. After that you can run the command to install :

`go install https://github.com/rubiin/projecto@latest`

# Usage

`projecto --add`
Adds the current directory as a new project. The editor used will be the global editor

`projecto --add --editor atom`
Adds the current directory as a new project. The editor used will be the atom


`projecto --rm`
Removes the project specified from the menu

`projecto --seteditor`
Sets the global editor from the list.If other is specified, a prompt is shown which required the command for the 
  editor (Eg. atom)

`projecto --open`
Opens a list where you can open the selected project


`projecto --rmeditor`
Removes editor for from the project.

`projecto --help`
Displays all the options that can be used




## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License
[MIT](https://choosealicense.com/licenses/mit/)
